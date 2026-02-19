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
				Metadata:   buildMetadata(item),
				Tags:       normalizeTags(item.Tags),
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
	} else if strings.TrimSpace(query.Symbol) != "" {
		q = q + " " + strings.TrimSpace(query.Symbol)
	}
	if q == "" {
		return nil, fmt.Errorf("recall: query is required")
	}

	reqBody := recallRequest{
		Query: q,
	}
	if len(query.Types) > 0 {
		reqBody.Types = filterHindsightTypes(query.Types)
	}
	if len(query.Tags) > 0 {
		reqBody.Tags = normalizeTags(query.Tags)
		reqBody.TagsMatch = "all_strict"
	}
	if query.To != nil {
		reqBody.QueryTimestamp = query.To.UTC().Format(time.RFC3339)
	} else if query.From != nil {
		reqBody.QueryTimestamp = query.From.UTC().Format(time.RFC3339)
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
		ts := parseTimestamp(r.MentionedAt, r.OccurredStart, r.OccurredEnd)
		data := buildDataFromRecall(r)
		itemType := derefString(r.Type, "memory")
		symbol := ""
		source := &contracts.MemorySource{System: "hindsight"}
		if meta := r.Metadata; meta != nil {
			if value := strings.TrimSpace(meta["item_type"]); value != "" {
				itemType = value
			}
			if value := strings.TrimSpace(meta["symbol"]); value != "" {
				symbol = value
			}
			if value := strings.TrimSpace(meta["source_system"]); value != "" {
				source.System = value
			}
			if value := strings.TrimSpace(meta["source_ref"]); value != "" {
				source.Ref = value
			}
		}
		item := contracts.MemoryItem{
			ID:      r.ID,
			TS:      ts,
			Type:    itemType,
			Symbol:  symbol,
			Tags:    normalizeTags(r.Tags),
			Summary: r.Text,
			Data:    data,
			Source:  source,
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

	queryText := strings.TrimSpace(params.Query)
	if hint := strings.TrimSpace(params.PromptHint); hint != "" {
		if queryText == "" {
			queryText = hint
		} else {
			queryText = queryText + "\n\n" + hint
		}
	}
	path := fmt.Sprintf("/v1/default/banks/%s/reflect", url.PathEscape(bank))
	reqBody := reflectRequest{
		Query: queryText,
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
	Content    string            `json:"content"`
	Context    string            `json:"context,omitempty"`
	DocumentID string            `json:"document_id,omitempty"`
	Timestamp  *string           `json:"timestamp,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	Tags       []string          `json:"tags,omitempty"`
}

type retainResponse struct {
	Success    bool   `json:"success"`
	BankID     string `json:"bank_id"`
	ItemsCount int    `json:"items_count"`
	Async      bool   `json:"async"`
}

type recallRequest struct {
	Query          string   `json:"query"`
	Types          []string `json:"types,omitempty"`
	QueryTimestamp string   `json:"query_timestamp,omitempty"`
	Tags           []string `json:"tags,omitempty"`
	TagsMatch      string   `json:"tags_match,omitempty"`
}

type recallResponse struct {
	Results []recallResult `json:"results"`
}

type recallResult struct {
	ID            string            `json:"id"`
	Text          string            `json:"text"`
	Type          *string           `json:"type"`
	Entities      []string          `json:"entities,omitempty"`
	Context       *string           `json:"context,omitempty"`
	OccurredStart *string           `json:"occurred_start,omitempty"`
	OccurredEnd   *string           `json:"occurred_end,omitempty"`
	MentionedAt   *string           `json:"mentioned_at,omitempty"`
	DocumentID    *string           `json:"document_id,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	ChunkID       *string           `json:"chunk_id,omitempty"`
	Tags          []string          `json:"tags,omitempty"`
}

type reflectRequest struct {
	Query string `json:"query"`
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

func buildMetadata(item contracts.MemoryItem) map[string]string {
	meta := map[string]string{}
	if value := strings.TrimSpace(item.Type); value != "" {
		meta["item_type"] = value
	}
	if value := strings.TrimSpace(item.Symbol); value != "" {
		meta["symbol"] = value
	}
	if item.Source != nil {
		if value := strings.TrimSpace(item.Source.System); value != "" {
			meta["source_system"] = value
		}
		if value := strings.TrimSpace(item.Source.Ref); value != "" {
			meta["source_ref"] = value
		}
	}
	if item.Data != nil {
		if raw, err := json.Marshal(item.Data); err == nil {
			meta["data_json"] = string(raw)
		}
	}
	if len(meta) == 0 {
		return nil
	}
	return meta
}

func normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		clean := strings.TrimSpace(tag)
		if clean == "" {
			continue
		}
		out = append(out, strings.ToLower(clean))
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func filterHindsightTypes(types []string) []string {
	if len(types) == 0 {
		return nil
	}
	out := make([]string, 0, len(types))
	seen := make(map[string]struct{}, len(types))
	for _, typ := range types {
		clean := strings.ToLower(strings.TrimSpace(typ))
		if clean == "" {
			continue
		}
		switch clean {
		case "world", "experience", "opinion":
		default:
			continue
		}
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		out = append(out, clean)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func parseTimestamp(values ...*string) time.Time {
	for _, value := range values {
		if value == nil {
			continue
		}
		ts := strings.TrimSpace(*value)
		if ts == "" {
			continue
		}
		parsed, err := time.Parse(time.RFC3339, ts)
		if err == nil {
			return parsed.UTC()
		}
	}
	return time.Now().UTC()
}

func buildDataFromRecall(r recallResult) map[string]any {
	data := map[string]any{}
	if len(r.Entities) > 0 {
		data["entities"] = r.Entities
	}
	if r.Context != nil && strings.TrimSpace(*r.Context) != "" {
		data["context"] = strings.TrimSpace(*r.Context)
	}
	if r.DocumentID != nil && strings.TrimSpace(*r.DocumentID) != "" {
		data["document_id"] = strings.TrimSpace(*r.DocumentID)
	}
	if r.ChunkID != nil && strings.TrimSpace(*r.ChunkID) != "" {
		data["chunk_id"] = strings.TrimSpace(*r.ChunkID)
	}
	if r.OccurredStart != nil && strings.TrimSpace(*r.OccurredStart) != "" {
		data["occurred_start"] = strings.TrimSpace(*r.OccurredStart)
	}
	if r.OccurredEnd != nil && strings.TrimSpace(*r.OccurredEnd) != "" {
		data["occurred_end"] = strings.TrimSpace(*r.OccurredEnd)
	}
	if r.MentionedAt != nil && strings.TrimSpace(*r.MentionedAt) != "" {
		data["mentioned_at"] = strings.TrimSpace(*r.MentionedAt)
	}

	meta := map[string]string{}
	for key, value := range r.Metadata {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			continue
		}
		meta[key] = value
	}
	if raw, ok := meta["data_json"]; ok {
		var parsed map[string]any
		if err := json.Unmarshal([]byte(raw), &parsed); err == nil {
			for key, value := range parsed {
				data[key] = value
			}
		}
		delete(meta, "data_json")
	}
	delete(meta, "item_type")
	delete(meta, "symbol")
	delete(meta, "source_system")
	delete(meta, "source_ref")
	if len(meta) > 0 {
		data["metadata"] = meta
	}

	return data
}
