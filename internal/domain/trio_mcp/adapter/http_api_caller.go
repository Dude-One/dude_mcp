package adapter

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/trio/triov2-mcp/internal/domain/trio_mcp/entity"
)

// HTTPAPICaller proxies HTTP calls to domain APIs.
// Auth is per-request — the caller passes the user's session token.
type HTTPAPICaller struct {
	client *http.Client
}

func NewHTTPAPICaller() *HTTPAPICaller {
	return &HTTPAPICaller{
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *HTTPAPICaller) Call(
	ctx context.Context,
	source entity.Source,
	method, path, tenantID, body, sessionToken string,
) ([]byte, int, error) {
	url := source.BaseURL + path

	var reqBody io.Reader
	if body != "" {
		reqBody = strings.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("building request %s %s: %w", method, url, err)
	}

	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// Inject user's session cookie for authenticated calls.
	if sessionToken != "" {
		req.AddCookie(&http.Cookie{Name: "trio_session", Value: sessionToken})
	}

	// Inject tenant scope header so the domain can enforce isolation.
	if tenantID != "" {
		req.Header.Set("X-Tenant-ID", tenantID)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("calling %s %s: %w", method, url, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("reading response body: %w", err)
	}

	return respBytes, resp.StatusCode, nil
}
