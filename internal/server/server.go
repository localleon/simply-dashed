package server

import (
	"embed"
	"html/template"
	"net/http"
	"strings"

	"github.com/localleon/simply-dashed/internal/config"
	"github.com/localleon/simply-dashed/internal/icons"
)

//go:embed templates/*.gohtml
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

type Server struct {
	cfg       *config.Config
	icons     *icons.Cache
	templates *template.Template
	version   string
}

type pageData struct {
	Title       string
	Query       string
	Groups      []groupView
	DisplayName string
	Version     string
}

type groupView struct {
	Name        string
	Description string
	Links       []linkView
}

type linkView struct {
	Name        string
	Description string
	URL         string
	IconPath    string
}

func New(cfg *config.Config, iconCache *icons.Cache, version string) (*Server, error) {
	tpl, err := template.ParseFS(templateFS, "templates/*.gohtml")
	if err != nil {
		return nil, err
	}
	return &Server{
		cfg:       cfg,
		icons:     iconCache,
		templates: tpl,
		version:   version,
	}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/search", s.handleSearch)
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.Handle("/icons/", http.StripPrefix("/icons/", http.FileServer(http.Dir(s.icons.Dir()))))
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	data := pageData{
		Title:       s.cfg.Title,
		Query:       query,
		Groups:      s.filterGroups(query),
		DisplayName: s.cfg.Title,
		Version:     s.version,
	}

	if err := s.templates.ExecuteTemplate(w, "index", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	data := pageData{
		Query:       query,
		Groups:      s.filterGroups(query),
		DisplayName: s.cfg.Title,
		Version:     s.version,
	}

	if err := s.templates.ExecuteTemplate(w, "results", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) filterGroups(query string) []groupView {
	query = strings.ToLower(strings.TrimSpace(query))
	var out []groupView

	for _, group := range s.cfg.Groups {
		view := groupView{
			Name:        group.Name,
			Description: group.Description,
		}

		for _, link := range group.Links {
			if matchesQuery(query, group, link) {
				view.Links = append(view.Links, linkView{
					Name:        link.Name,
					Description: link.Description,
					URL:         link.URL,
					IconPath:    s.icons.Resolve(link.Icon),
				})
			}
		}

		if len(view.Links) > 0 {
			out = append(out, view)
		}
	}

	return out
}

func matchesQuery(query string, group config.Group, link config.Link) bool {
	if query == "" {
		return true
	}
	fields := []string{
		group.Name,
		group.Description,
		link.Name,
		link.Description,
		link.URL,
	}
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), query) {
			return true
		}
	}
	return false
}
