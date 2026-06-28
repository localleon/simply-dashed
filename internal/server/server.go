package server

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"
	"net/url"
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
	TotalLinks  int
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
	staticFiles, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err)
	}
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFiles))))
	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	data := s.newPageData(query)
	data.Title = s.cfg.Title

	if err := s.templates.ExecuteTemplate(w, "index", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Push-Url", searchPageURL(query))
	}
	data := s.newPageData(query)

	if err := s.templates.ExecuteTemplate(w, "results", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) newPageData(query string) pageData {
	groups, totalLinks := s.filterGroups(query)
	return pageData{
		Query:       query,
		Groups:      groups,
		TotalLinks:  totalLinks,
		DisplayName: s.cfg.Title,
		Version:     s.version,
	}
}

func (s *Server) filterGroups(query string) ([]groupView, int) {
	query = strings.ToLower(strings.TrimSpace(query))
	var out []groupView
	totalLinks := 0

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
				totalLinks++
			}
		}

		if len(view.Links) > 0 {
			out = append(out, view)
		}
	}

	return out, totalLinks
}

func searchPageURL(query string) string {
	if query == "" {
		return "/"
	}
	return "/?q=" + url.QueryEscape(query)
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
