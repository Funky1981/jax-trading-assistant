package hindsight

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"jax-trading-assistant/libs/contracts"
)

type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
}

type Option func(*Client)

func WithHTTPClient(c *http.Client) Option {
	return func(cl *Client) {
		cl.httpClient = c
	}
}

func New(baseURL string, opts ...Option) (*Client, error) {
	u, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return nil, fmt.Errorf("hindsight client: parse base url: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("hindsight client: base url must include scheme and host")
	}

	c := &Client{
		baseURL: u,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

var _ contracts.MemoryStore = (*Client)(nil)

func (c *Client) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.resolve("/health"), nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
		return fmt.Errorf("hindsight ping failed: %s", strings.TrimSpace(string(body)))
	}
	return nil
}

func (c *Client) Retain(ctx context.Context, bank string, item contracts.MemoryItem) (contracts.MemoryID, error) {
	path := fmt.Sprintf("/v1/default/banks/%s/memories", url.PathEscape(bank))

	content := item.Summary
	if content == "" {
		content = "memory item"
	}

	contextStr := ""
	if item.Type != "" || item.Symbol != "" || len(item.Tags) > 0 {
		contextStr = fmt.Sprintf("type=%s symbol=%s tags=%s", item.Type, item.Symbol, strings.Join(item.Tags, ","))
	}

	var ts *string
	if !item.TS.IsZero() {
		s := item.TS.UTC().Format(time.RFC3339)
		ts = &s
	}

	docID := item.ID
	if docID == "" {
		docID = fmt.Sprintf("jax_%d", time.Now().UTC().UnixNano())
	}

	reqBody := retainRequest{
		Async: false,
		Items: []retainItem{
			{
				Content:    content,
				Context:    contextStr,
				DocumentID: docID,
				Timestamp:  ts,
			},
		},
	}

	if err := c.postJSON(ctx, path, reqBody, &retainResponse{}); err != nil {
		return "", err
	}

	return contracts.MemoryID(docID), nil
}

func (c *Client) Recall(ctx context.Context, bank string, query contracts.MemoryQuery) ([]contracts.MemoryItem, error) {
	path := fmt.Sprintf("/v1/default/banks/%s/memories/recall", url.PathEscape(bank))

	q := strings.TrimSpace(query.Q)
	if q == "" {
		parts := []string{}
		if query.Symbol != "" {
			parts = append(parts, query.Symbol)
		}
		parts = append(parts, query.Tags...)
		q = strings.Join(parts, " ")
	}
	if q == "" {
		return nil, fmt.Errorf("recall: query is required")
	}

	reqBody := recallRequest{
		Query: q,
	}
	if len(query.Types) > 0 {
		reqBody.Types = query.Types
	}

	var resp recallResponse
	if err := c.postJSON(ctx, path, reqBody, &resp); err != nil {
		return nil, err
	}

	limit := query.Limit
	if limit <= 0 {
		limit = 100
	}

	out := make([]contracts.MemoryItem, 0, len(resp.Results))
	for _, r := range resp.Results {
		item := contracts.MemoryItem{
			ID:      r.ID,
			TS:      time.Now().UTC(),
			Type:    derefString(r.Type, "memory"),
			Summary: r.Text,
			Data: map[string]any{
				"entities": r.Entities,
			},
			Source: &contracts.MemorySource{System: "hindsight"},
		}
		out = append(out, item)
		if len(out) >= limit {
			break
		}
	}

	return out, nil
}

func (c *Client) Reflect(ctx context.Context, bank string, params contracts.ReflectionParams) ([]contracts.MemoryItem, error) {
	if strings.TrimSpace(params.Query) == "" {
		return nil, fmt.Errorf("reflect: params.query is required")
	}

	path := fmt.Sprintf("/v1/default/banks/%s/reflect", url.PathEscape(bank))
	reqBody := reflectRequest{
		Query:   params.Query,
		Context: params.PromptHint,
	}

	var resp reflectResponse
	if err := c.postJSON(ctx, path, reqBody, &resp); err != nil {
		return nil, err
	}

	return []contracts.MemoryItem{
		{
			TS:      time.Now().UTC(),
			Type:    "belief",
			Summary: resp.Text,
			Data: map[string]any{
				"based_on": resp.BasedOn,
			},
			Source: &contracts.MemorySource{System: "hindsight"},
		},
	}, nil
}

func (c *Client) resolve(path string) string {
	u := *c.baseURL
	u.Path = strings.TrimSuffix(u.Path, "/") + path
	return u.String()
}

func (c *Client) postJSON(ctx context.Context, path string, input any, output any) error {
	body, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("hindsight: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.resolve(path), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("hindsight: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("hindsight: http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("hindsight: read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("hindsight: http %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	if output == nil {
		return nil
	}
	if err := json.Unmarshal(respBody, output); err != nil {
		return fmt.Errorf("hindsight: decode response: %w", err)
	}
	return nil
}

type retainRequest struct {
	Items []retainItem `json:"items"`
	Async bool         `json:"async"`
}

type retainItem struct {
	Content    string  `json:"content"`
	Context    string  `json:"context,omitempty"`
	DocumentID string  `json:"document_id,omitempty"`
	Timestamp  *string `json:"timestamp,omitempty"`
}

type retainResponse struct {
	Success    bool   `json:"success"`
	BankID     string `json:"bank_id"`
	ItemsCount int    `json:"items_count"`
	Async      bool   `json:"async"`
}

type recallRequest struct {
	Query string   `json:"query"`
	Types []string `json:"types,omitempty"`
}

type recallResponse struct {
	Results []recallResult `json:"results"`
}

type recallResult struct {
	ID       string    `json:"id"`
	Text     string    `json:"text"`
	Type     *string   `json:"type"`
	Entities *[]string `json:"entities"`
}

type reflectRequest struct {
	Query   string `json:"query"`
	Context string `json:"context,omitempty"`
}

type reflectResponse struct {
	Text    string          `json:"text"`
	BasedOn json.RawMessage `json:"based_on,omitempty"`
}

func derefString(v *string, fallback string) string {
	if v == nil || *v == "" {
		return fallback
	}
	return *v
}
