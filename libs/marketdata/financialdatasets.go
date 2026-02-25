package marketdata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// FinancialDatasetsProvider implements market data via financialdatasets.ai APIs.
type FinancialDatasetsProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func NewFinancialDatasetsProvider(cfg ProviderConfig) (*FinancialDatasetsProvider, error) {
	key := strings.TrimSpace(cfg.APIKey)
	if key == "" {
		return nil, errors.New("financial-datasets API key is required")
	}
	return &FinancialDatasetsProvider{
		apiKey:  key,
		baseURL: "https://api.financialdatasets.ai",
		client:  &http.Client{Timeout: 15 * time.Second},
	}, nil
}

func (p *FinancialDatasetsProvider) Name() string { return string(ProviderFinancialDatasets) }

func (p *FinancialDatasetsProvider) GetQuote(ctx context.Context, symbol string) (*Quote, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" {
		return nil, errors.New("symbol is required")
	}
	endpoint := p.baseURL + "/prices/snapshot"
	u, _ := url.Parse(endpoint)
	q := u.Query()
	q.Set("ticker", symbol)
	u.RawQuery = q.Encode()

	var payload struct {
		Snapshot struct {
			Price float64 `json:"price"`
			Time  string  `json:"time"`
		} `json:"snapshot"`
		Price float64 `json:"price"`
		Time  string  `json:"time"`
	}
	if err := p.fetchJSON(ctx, u.String(), &payload); err != nil {
		return nil, err
	}
	price := payload.Price
	ts := payload.Time
	if payload.Snapshot.Price != 0 || payload.Snapshot.Time != "" {
		price = payload.Snapshot.Price
		ts = payload.Snapshot.Time
	}
	if price == 0 {
		return nil, fmt.Errorf("financial-datasets snapshot missing price")
	}
	parsedTS, _ := parseFDTime(ts)
	if parsedTS.IsZero() {
		parsedTS = time.Now().UTC()
	}
	return &Quote{
		Symbol:    symbol,
		Price:     price,
		Timestamp: parsedTS.UTC(),
	}, nil
}

func (p *FinancialDatasetsProvider) GetCandles(ctx context.Context, symbol string, timeframe Timeframe, limit int) ([]Candle, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" {
		return nil, errors.New("symbol is required")
	}
	interval, multiplier := mapTimeframe(timeframe)
	if interval == "" {
		return nil, fmt.Errorf("unsupported timeframe %q", timeframe)
	}
	if limit <= 0 {
		limit = 200
	}
	endpoint := p.baseURL + "/prices/"
	u, _ := url.Parse(endpoint)
	q := u.Query()
	q.Set("ticker", symbol)
	q.Set("interval", interval)
	q.Set("interval_multiplier", strconv.Itoa(multiplier))
	q.Set("limit", strconv.Itoa(limit))
	u.RawQuery = q.Encode()

	var payload struct {
		Ticker string `json:"ticker"`
		Prices []struct {
			Time   string  `json:"time"`
			Open   float64 `json:"open"`
			High   float64 `json:"high"`
			Low    float64 `json:"low"`
			Close  float64 `json:"close"`
			Volume int64   `json:"volume"`
		} `json:"prices"`
	}
	if err := p.fetchJSON(ctx, u.String(), &payload); err != nil {
		return nil, err
	}
	out := make([]Candle, 0, len(payload.Prices))
	for _, row := range payload.Prices {
		ts, _ := parseFDTime(row.Time)
		if ts.IsZero() {
			continue
		}
		out = append(out, Candle{
			Symbol:    symbol,
			Timestamp: ts.UTC(),
			Open:      row.Open,
			High:      row.High,
			Low:       row.Low,
			Close:     row.Close,
			Volume:    row.Volume,
		})
	}
	return out, nil
}

func (p *FinancialDatasetsProvider) GetTrades(ctx context.Context, symbol string, limit int) ([]Trade, error) {
	return nil, errors.New("financial-datasets does not support trades in this client")
}

func (p *FinancialDatasetsProvider) GetEarnings(ctx context.Context, symbol string, limit int) ([]Earnings, error) {
	return nil, errors.New("financial-datasets does not support earnings in this client")
}

func (p *FinancialDatasetsProvider) StreamQuotes(ctx context.Context, symbols []string) (<-chan StreamUpdate, error) {
	return nil, errors.New("financial-datasets does not support streaming")
}

func (p *FinancialDatasetsProvider) HealthCheck(ctx context.Context) error {
	if strings.TrimSpace(p.apiKey) == "" {
		return errors.New("financial-datasets API key missing")
	}
	return nil
}

func (p *FinancialDatasetsProvider) fetchJSON(ctx context.Context, endpoint string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-API-KEY", p.apiKey)
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("financial-datasets HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(out)
}

func mapTimeframe(tf Timeframe) (string, int) {
	switch tf {
	case Timeframe1Min:
		return "minute", 1
	case Timeframe5Min:
		return "minute", 5
	case Timeframe15Min:
		return "minute", 15
	case Timeframe1Hour:
		return "hour", 1
	case Timeframe1Day:
		return "day", 1
	case Timeframe1Week:
		return "week", 1
	default:
		return "", 0
	}
}

func parseFDTime(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, nil
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported time %q", raw)
}
