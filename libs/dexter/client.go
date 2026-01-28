package dexter

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
)

// Client wraps Dexter tools server for research operations
type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
}

// Option configures the Dexter client
type Option func(*Client)

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(c *http.Client) Option {
	return func(cl *Client) {
		cl.httpClient = c
	}
}

// New creates a new Dexter client
func New(baseURL string, opts ...Option) (*Client, error) {
	u, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return nil, fmt.Errorf("dexter client: parse base url: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("dexter client: base url must include scheme and host")
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

// ResearchCompanyInput contains inputs for company research
type ResearchCompanyInput struct {
	Ticker    string   `json:"ticker"`
	Questions []string `json:"questions"`
}

// ResearchCompanyOutput contains company research results
type ResearchCompanyOutput struct {
	Ticker      string                 `json:"ticker"`
	Summary     string                 `json:"summary"`
	KeyPoints   []string               `json:"key_points"`
	Metrics     map[string]interface{} `json:"metrics"`
	RawMarkdown string                 `json:"raw_markdown"`
}

// ResearchCompany performs deep research on a company
func (c *Client) ResearchCompany(ctx context.Context, input ResearchCompanyInput) (ResearchCompanyOutput, error) {
	req := toolRequest{
		Tool:  "dexter.research_company",
		Input: input,
	}

	var resp toolResponse
	if err := c.callTool(ctx, req, &resp); err != nil {
		return ResearchCompanyOutput{}, err
	}

	var output ResearchCompanyOutput
	if err := mapToStruct(resp.Output, &output); err != nil {
		return ResearchCompanyOutput{}, fmt.Errorf("decode research output: %w", err)
	}

	return output, nil
}

// CompareCompaniesInput contains inputs for company comparison
type CompareCompaniesInput struct {
	Tickers []string `json:"tickers"`
	Focus   string   `json:"focus"`
}

// ComparisonItem represents one company in a comparison
type ComparisonItem struct {
	Ticker string   `json:"ticker"`
	Thesis string   `json:"thesis"`
	Notes  []string `json:"notes"`
}

// CompareCompaniesOutput contains comparison results
type CompareCompaniesOutput struct {
	ComparisonAxis string           `json:"comparison_axis"`
	Items          []ComparisonItem `json:"items"`
}

// CompareCompanies compares multiple companies on a specific dimension
func (c *Client) CompareCompanies(ctx context.Context, input CompareCompaniesInput) (CompareCompaniesOutput, error) {
	req := toolRequest{
		Tool:  "dexter.compare_companies",
		Input: input,
	}

	var resp toolResponse
	if err := c.callTool(ctx, req, &resp); err != nil {
		return CompareCompaniesOutput{}, err
	}

	var output CompareCompaniesOutput
	if err := mapToStruct(resp.Output, &output); err != nil {
		return CompareCompaniesOutput{}, fmt.Errorf("decode comparison output: %w", err)
	}

	return output, nil
}

// Health checks if the Dexter tools server is available
func (c *Client) Health(ctx context.Context) error {
	endpoint := c.baseURL.ResolveReference(&url.URL{Path: "/health"})

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("health check: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("dexter unhealthy: status=%d", resp.StatusCode)
	}

	return nil
}

// Internal types for API communication

type toolRequest struct {
	Tool  string      `json:"tool"`
	Input interface{} `json:"input"`
}

type toolResponse struct {
	Output interface{} `json:"output,omitempty"`
	Error  string      `json:"error,omitempty"`
}

func (c *Client) callTool(ctx context.Context, req toolRequest, resp *toolResponse) error {
	endpoint := c.baseURL.ResolveReference(&url.URL{Path: "/tools"})

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer httpResp.Body.Close()

	bodyBytes, err := io.ReadAll(io.LimitReader(httpResp.Body, 1<<20)) // 1MB limit
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return fmt.Errorf("dexter tool failed: status=%d body=%s", httpResp.StatusCode, string(bodyBytes))
	}

	if err := json.Unmarshal(bodyBytes, resp); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if resp.Error != "" {
		return fmt.Errorf("dexter error: %s", resp.Error)
	}

	return nil
}

// mapToStruct converts map[string]interface{} to a specific struct
func mapToStruct(input interface{}, output interface{}) error {
	data, err := json.Marshal(input)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, output)
}
