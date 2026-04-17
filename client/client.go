package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/adortb/adortb-dco/internal/engine"
)

// Client is the DCO service client for use by DSP.
type Client struct {
	baseURL string
	http    *http.Client
}

// New creates a new DCO client with sensible defaults.
func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http: &http.Client{
			Timeout: 50 * time.Millisecond,
		},
	}
}

// Render calls POST /v1/render and returns the assembled creative.
func (c *Client) Render(ctx context.Context, req engine.RenderRequest) (*engine.RenderResult, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("dco client marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/render", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("dco client new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("dco client do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		return nil, fmt.Errorf("dco render http %d: %s", resp.StatusCode, errBody["error"])
	}

	var result engine.RenderResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("dco client decode: %w", err)
	}
	return &result, nil
}
