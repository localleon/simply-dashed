package icons

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/localleon/simply-dashed/internal/config"
)

type Cache struct {
	dir    string
	client *http.Client

	mu    sync.RWMutex
	paths map[string]string
}

func NewCache(dir string) (*Cache, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", dir, err)
	}

	return &Cache{
		dir: dir,
		client: &http.Client{
			Timeout: 20 * time.Second,
		},
		paths: make(map[string]string),
	}, nil
}

func (c *Cache) SetHTTPClient(client *http.Client) {
	if client != nil {
		c.client = client
	}
}

func (c *Cache) Prime(ctx context.Context, cfg *config.Config, refresh bool) error {
	var errs []string
	for _, group := range cfg.Groups {
		for _, link := range group.Links {
			if strings.TrimSpace(link.Icon) == "" {
				continue
			}
			if _, err := c.Download(ctx, link.Icon, refresh); err != nil {
				errs = append(errs, fmt.Sprintf("%s: %v", link.Name, err))
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

func (c *Cache) Download(ctx context.Context, source string, refresh bool) (string, error) {
	filename := iconFilename(source, "")
	target := filepath.Join(c.dir, filename)

	c.mu.RLock()
	if path, ok := c.paths[source]; ok {
		c.mu.RUnlock()
		return path, nil
	}
	c.mu.RUnlock()

	if _, err := os.Stat(target); err == nil {
		relPath := "/icons/" + filename
		c.mu.Lock()
		c.paths[source] = relPath
		c.mu.Unlock()
		if !refresh {
			return relPath, nil
		}
	}

	if !refresh {
		return "", fmt.Errorf("icon missing in cache and refresh disabled")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return "", fmt.Errorf("new request: %w", err)
	}

	res, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", fmt.Errorf("unexpected status: %s", res.Status)
	}

	body, err := io.ReadAll(io.LimitReader(res.Body, 5<<20))
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}

	filename = iconFilename(source, res.Header.Get("Content-Type"))
	target = filepath.Join(c.dir, filename)

	if err := os.WriteFile(target, body, 0o644); err != nil {
		return "", fmt.Errorf("write icon: %w", err)
	}

	relPath := "/icons/" + filename
	c.mu.Lock()
	c.paths[source] = relPath
	c.mu.Unlock()

	return relPath, nil
}

func (c *Cache) Resolve(source string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.paths[source]
}

func (c *Cache) Dir() string {
	return c.dir
}

func iconFilename(source, contentType string) string {
	sum := sha256.Sum256([]byte(source))
	ext := extensionFromContentType(contentType)
	if ext == "" {
		if parsed, err := url.Parse(source); err == nil {
			ext = filepath.Ext(parsed.Path)
		}
	}
	if ext == "" {
		ext = ".img"
	}

	return hex.EncodeToString(sum[:12]) + ext
}

func extensionFromContentType(contentType string) string {
	if contentType == "" {
		return ""
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return ""
	}
	switch mediaType {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/svg+xml":
		return ".svg"
	case "image/webp":
		return ".webp"
	case "image/x-icon", "image/vnd.microsoft.icon":
		return ".ico"
	default:
		return ""
	}
}
