package adapter

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/open-mcp/open-mcp/internal/domain/open_mcp/entity"
)

// swaggerCacheEntry holds a fetched spec with its expiry time.
type swaggerCacheEntry struct {
	data      []byte
	expiresAt time.Time
}

// HTTPSwaggerReader fetches and TTL-caches Swagger specs from domain services.
// Cache is in-process (soft/per-replica) — per-replica divergence is acceptable.
// Move to Redis if cross-replica freshness consistency is required (Phase 4+).
type HTTPSwaggerReader struct {
	client *http.Client
	ttl    time.Duration

	mu    sync.Mutex
	cache map[string]swaggerCacheEntry // keyed by source name
}

func NewHTTPSwaggerReader(ttl time.Duration) *HTTPSwaggerReader {
	return &HTTPSwaggerReader{
		client: &http.Client{Timeout: 10 * time.Second},
		ttl:    ttl,
		cache:  make(map[string]swaggerCacheEntry),
	}
}

func (r *HTTPSwaggerReader) GetSpec(ctx context.Context, source entity.Source) ([]byte, error) {
	r.mu.Lock()
	entry, ok := r.cache[source.Name]
	if ok && time.Now().Before(entry.expiresAt) {
		r.mu.Unlock()
		return entry.data, nil
	}
	r.mu.Unlock()

	url := source.BaseURL + source.SwaggerPath
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("building swagger request for %q: %w", source.Name, err)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching swagger for %q: %w", source.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("swagger fetch for %q returned status %d", source.Name, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading swagger body for %q: %w", source.Name, err)
	}

	r.mu.Lock()
	r.cache[source.Name] = swaggerCacheEntry{
		data:      data,
		expiresAt: time.Now().Add(r.ttl),
	}
	r.mu.Unlock()

	return data, nil
}
