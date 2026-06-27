package icons

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDownloadCachesAndReusesLocalFile(t *testing.T) {
	t.Parallel()

	var requests int
	source := "https://icons.example.com/icon.png"
	dir := t.TempDir()

	cache, err := NewCache(dir)
	if err != nil {
		t.Fatalf("new cache: %v", err)
	}
	cache.SetHTTPClient(&http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			requests++
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     http.Header{"Content-Type": []string{"image/png"}},
				Body:       io.NopCloser(strings.NewReader("png-data")),
				Request:    r,
			}, nil
		}),
	})

	path, err := cache.Download(context.Background(), source, true)
	if err != nil {
		t.Fatalf("download icon: %v", err)
	}
	if !strings.HasPrefix(path, "/icons/") {
		t.Fatalf("path = %q, want /icons/*", path)
	}
	if cache.Resolve(source) != path {
		t.Fatalf("resolve = %q, want %q", cache.Resolve(source), path)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}

	filename := strings.TrimPrefix(path, "/icons/")
	if _, err := os.Stat(filepath.Join(dir, filename)); err != nil {
		t.Fatalf("stat cached icon: %v", err)
	}

	cache2, err := NewCache(dir)
	if err != nil {
		t.Fatalf("new second cache: %v", err)
	}

	path2, err := cache2.Download(context.Background(), source, false)
	if err != nil {
		t.Fatalf("reuse cached icon: %v", err)
	}
	if path2 != path {
		t.Fatalf("path2 = %q, want %q", path2, path)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want cache hit without new fetch", requests)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}
