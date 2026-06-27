package server

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/localleon/simply-dashed/internal/config"
	"github.com/localleon/simply-dashed/internal/icons"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()

	iconSource := "https://icons.example.com/grafana.png"
	cache, err := icons.NewCache(t.TempDir())
	if err != nil {
		t.Fatalf("new cache: %v", err)
	}
	cache.SetHTTPClient(&http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     http.Header{"Content-Type": []string{"image/png"}},
				Body:       io.NopCloser(strings.NewReader("png-data")),
				Request:    r,
			}, nil
		}),
	})
	if _, err := cache.Download(context.Background(), iconSource, true); err != nil {
		t.Fatalf("download icon: %v", err)
	}

	cfg := &config.Config{
		Title: "Links",
		Groups: []config.Group{
			{
				Name:        "Infrastructure",
				Description: "Ops tools",
				Links: []config.Link{
					{
						Name:        "Grafana",
						Description: "Dashboards",
						URL:         "https://grafana.example.com",
						Icon:        iconSource,
					},
					{
						Name:        "Argo CD",
						Description: "Deployments",
						URL:         "https://argo.example.com",
					},
				},
			},
		},
	}

	srv, err := New(cfg, cache, "v1.2.3")
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	return srv
}

func TestHandleIndexRendersFooterAndAssets(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()
	for _, want := range []string{
		"/static/vendor/htmx-2.0.10.min.js",
		`hx-trigger="keyup changed delay:120ms, search"`,
		`<footer class="footer">Links <span>·</span> v1.2.3</footer>`,
		`/icons/`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q", want)
		}
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}

func TestHandleSearchFiltersResults(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/search?q=gra", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Grafana") {
		t.Fatalf("body missing Grafana result")
	}
	if strings.Contains(body, "Argo CD") {
		t.Fatalf("body unexpectedly contains non-matching result")
	}
}

func TestHealthEndpoint(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if strings.TrimSpace(rec.Body.String()) != "ok" {
		t.Fatalf("body = %q, want %q", rec.Body.String(), "ok")
	}
}
