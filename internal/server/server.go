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
	Subtitle    string
	Query       string
	Groups      []groupView
	TotalLinks  int
	DisplayName string
	Version     string
}

type dashboardIndexData struct {
	Title      string
	Subtitle   string
	Dashboards []dashboardSummary
	Version    string
}

type dashboardSummary struct {
	Title    string
	Subtitle string
	Path     string
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
	if r.URL.Path == "/" {
		s.handleDashboardIndex(w, r)
		return
	}

	dashboard, remainder, ok := s.matchDashboard(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}

	switch remainder {
	case "":
		http.Redirect(w, r, dashboard.Path+"/"+querySuffix(r.URL.RawQuery), http.StatusMovedPermanently)
	case "/":
		s.handleDashboardPage(w, r, dashboard)
	case "/search":
		s.handleSearch(w, r, dashboard)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleDashboardIndex(w http.ResponseWriter, r *http.Request) {
	data := dashboardIndexData{
		Title:      s.cfg.Title,
		Subtitle:   s.cfg.Subtitle,
		Dashboards: s.dashboardSummaries(),
		Version:    s.version,
	}

	if err := s.templates.ExecuteTemplate(w, "dashboards", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleDashboardPage(w http.ResponseWriter, r *http.Request, dashboard *config.Dashboard) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	data := s.newPageData(dashboard, query)

	if err := s.templates.ExecuteTemplate(w, "index", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request, dashboard *config.Dashboard) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Push-Url", searchPageURL(dashboard.Path, query))
		data := s.newPageData(dashboard, query)

		if err := s.templates.ExecuteTemplate(w, "results", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	data := s.newPageData(dashboard, query)

	if err := s.templates.ExecuteTemplate(w, "index", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) newPageData(dashboard *config.Dashboard, query string) pageData {
	groups, totalLinks := s.filterGroups(dashboard.Groups, query)
	return pageData{
		Title:       dashboard.Title,
		Subtitle:    dashboard.Subtitle,
		Query:       query,
		Groups:      groups,
		TotalLinks:  totalLinks,
		DisplayName: dashboard.Title,
		Version:     s.version,
	}
}

func (s *Server) dashboardSummaries() []dashboardSummary {
	out := make([]dashboardSummary, 0, len(s.cfg.Dashboards))
	for _, dashboard := range s.cfg.Dashboards {
		out = append(out, dashboardSummary{
			Title:    dashboard.Title,
			Subtitle: dashboard.Subtitle,
			Path:     dashboard.Path + "/",
		})
	}
	return out
}

func (s *Server) filterGroups(groups []config.Group, query string) ([]groupView, int) {
	query = strings.ToLower(strings.TrimSpace(query))
	var out []groupView
	totalLinks := 0

	for _, group := range groups {
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

func (s *Server) matchDashboard(path string) (*config.Dashboard, string, bool) {
	var match *config.Dashboard
	longest := -1
	var remainder string
	for i := range s.cfg.Dashboards {
		dashboard := &s.cfg.Dashboards[i]
		if path == dashboard.Path || strings.HasPrefix(path, dashboard.Path+"/") {
			if len(dashboard.Path) > longest {
				match = dashboard
				longest = len(dashboard.Path)
				remainder = strings.TrimPrefix(path, dashboard.Path)
			}
		}
	}
	if match == nil {
		return nil, "", false
	}
	return match, remainder, true
}

func searchPageURL(basePath, query string) string {
	if basePath == "" {
		basePath = "/"
	}
	if query == "" {
		return basePath + "/"
	}
	return basePath + "/?q=" + url.QueryEscape(query)
}

func querySuffix(raw string) string {
	if raw == "" {
		return ""
	}
	return "?" + raw
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
