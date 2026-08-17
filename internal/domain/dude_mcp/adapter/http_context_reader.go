package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Dude-One/dude_mcp/internal/domain/dude_mcp/entity"
)

// HTTPContextReader fetches context docs from a domain's /mcp/context endpoints.
type HTTPContextReader struct {
	client *http.Client
}

func NewHTTPContextReader() *HTTPContextReader {
	return &HTTPContextReader{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (r *HTTPContextReader) ListDocs(ctx context.Context, source entity.Source) ([]string, error) {
	url := source.BaseURL + source.ContextPath
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("building context list request for %q: %w", source.Name, err)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("listing context docs for %q: %w", source.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("context list for %q returned status %d", source.Name, resp.StatusCode)
	}

	var paths []string
	if err := json.NewDecoder(resp.Body).Decode(&paths); err != nil {
		return nil, fmt.Errorf("decoding context list for %q: %w", source.Name, err)
	}
	return paths, nil
}

func (r *HTTPContextReader) ReadDoc(ctx context.Context, source entity.Source, path string) ([]byte, error) {
	url := source.BaseURL + source.ContextPath + "/" + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("building context read request for %q/%s: %w", source.Name, path, err)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("reading context doc %q/%s: %w", source.Name, path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("context doc %q not found in source %q", path, source.Name)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("context read for %q/%s returned status %d", source.Name, path, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading context body for %q/%s: %w", source.Name, path, err)
	}
	return data, nil
}
