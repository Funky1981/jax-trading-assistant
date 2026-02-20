// Package dataset provides L03: versioned dataset management with content-hash
// reproducibility.  Datasets are OHLCV CSV files catalogued in a JSON registry
// file.  A CSVDataSource adapts any registered dataset into the
// strategies.HistoricalDataSource interface required by the backtest engine.
package dataset

import (
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"jax-trading-assistant/libs/strategies"
)

// ─── Schema version ───────────────────────────────────────────────────────────

const schemaVer = "ohlcv_v1"

// ─── Dataset ──────────────────────────────────────────────────────────────────

// Dataset describes one catalogued data file.
type Dataset struct {
	// ID is a UUID assigned by Register.
	ID string `json:"id"`
	// Name is a human-readable label e.g. "AAPL_2023".
	Name string `json:"name"`
	// Symbol is the primary ticker e.g. "AAPL".
	Symbol string `json:"symbol"`
	// Source describes the origin: "csv", "ib", "alpha_vantage", etc.
	Source string `json:"source"`
	// StartDate / EndDate are the inclusive date range of the data.
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	// FilePath is the path to the OHLCV CSV file (absolute or relative to CWD).
	FilePath string `json:"file_path"`
	// Hash is the SHA-256 hex digest of the file content at registration time.
	// Use this to detect file mutations that would break determinism.
	Hash string `json:"hash"`
	// SchemaVer is the CSV schema version string.
	SchemaVer string `json:"schema_ver"`
	// CreatedAt is when Register() was called.
	CreatedAt time.Time `json:"created_at"`
	// RecordCount is the number of candle rows found in the file.
	RecordCount int `json:"record_count"`
}

// ─── Registry ─────────────────────────────────────────────────────────────────

const catalogFile = "catalog.json"

// Registry is a thread-safe store of Dataset records persisted as JSON in a
// directory on disk.
type Registry struct {
	mu         sync.RWMutex
	catalogDir string
	datasets   map[string]Dataset // keyed by ID
}

// Open loads (or creates) a Registry backed by catalogDir.
// The directory is created if it does not exist.
func Open(catalogDir string) (*Registry, error) {
	if err := os.MkdirAll(catalogDir, 0o755); err != nil {
		return nil, fmt.Errorf("dataset.Open: mkdir %q: %w", catalogDir, err)
	}

	r := &Registry{
		catalogDir: catalogDir,
		datasets:   make(map[string]Dataset),
	}

	if err := r.load(); err != nil {
		return nil, err
	}

	return r, nil
}

// Register validates the CSV file at d.FilePath, computes its SHA-256 hash,
// assigns a UUID, and persists the entry to the catalog.
// An error is returned if the file does not exist or has a duplicate Name.
func (r *Registry) Register(d Dataset) (Dataset, error) {
	if d.Name == "" {
		return Dataset{}, fmt.Errorf("dataset.Register: Name must not be empty")
	}
	if d.Symbol == "" {
		return Dataset{}, fmt.Errorf("dataset.Register: Symbol must not be empty")
	}
	if d.FilePath == "" {
		return Dataset{}, fmt.Errorf("dataset.Register: FilePath must not be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Reject duplicate names.
	for _, existing := range r.datasets {
		if existing.Name == d.Name {
			return Dataset{}, fmt.Errorf("dataset.Register: name %q already registered (id=%s)", d.Name, existing.ID)
		}
	}

	// Hash and stat the file.
	hash, count, err := hashAndCount(d.FilePath)
	if err != nil {
		return Dataset{}, fmt.Errorf("dataset.Register: file %q: %w", d.FilePath, err)
	}

	d.ID = uuid.New().String()
	d.Hash = hash
	d.RecordCount = count
	d.SchemaVer = schemaVer
	d.CreatedAt = time.Now().UTC()
	if d.Source == "" {
		d.Source = "csv"
	}

	r.datasets[d.ID] = d

	if err := r.save(); err != nil {
		delete(r.datasets, d.ID)
		return Dataset{}, fmt.Errorf("dataset.Register: persist: %w", err)
	}

	log.Printf("[dataset] registered name=%q id=%s symbol=%s records=%d hash=%s",
		d.Name, d.ID, d.Symbol, d.RecordCount, d.Hash[:12])

	return d, nil
}

// Get returns the Dataset with the given ID.
func (r *Registry) Get(id string) (Dataset, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	d, ok := r.datasets[id]
	if !ok {
		return Dataset{}, fmt.Errorf("dataset.Get: id %q not found", id)
	}
	return d, nil
}

// GetByName returns the first Dataset whose Name matches.
func (r *Registry) GetByName(name string) (Dataset, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, d := range r.datasets {
		if d.Name == name {
			return d, nil
		}
	}
	return Dataset{}, fmt.Errorf("dataset.GetByName: %q not found", name)
}

// List returns all Datasets sorted by CreatedAt ascending.
func (r *Registry) List() []Dataset {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]Dataset, 0, len(r.datasets))
	for _, d := range r.datasets {
		out = append(out, d)
	}
	slices.SortFunc(out, func(a, b Dataset) int {
		return a.CreatedAt.Compare(b.CreatedAt)
	})
	return out
}

// Remove deletes a Dataset entry from the catalog.  It does NOT delete the
// underlying data file.
func (r *Registry) Remove(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.datasets[id]; !ok {
		return fmt.Errorf("dataset.Remove: id %q not found", id)
	}
	delete(r.datasets, id)
	return r.save()
}

// VerifyHash re-computes the file hash and returns an error if it has changed
// since registration (which would invalidate backtest reproducibility).
func (r *Registry) VerifyHash(id string) error {
	d, err := r.Get(id)
	if err != nil {
		return err
	}

	hash, _, err := hashAndCount(d.FilePath)
	if err != nil {
		return fmt.Errorf("dataset.VerifyHash: %w", err)
	}

	if hash != d.Hash {
		return fmt.Errorf("dataset.VerifyHash: id=%s file content has changed (registered=%s current=%s)",
			id, d.Hash[:12], hash[:12])
	}
	return nil
}

// LoadDataSource opens a registered CSV dataset as a strategies.HistoricalDataSource
// ready for use by the backtest engine.  The file hash is NOT re-verified here
// for performance; call VerifyHash first if strict reproducibility is required.
func (r *Registry) LoadDataSource(_ context.Context, id string) (*CSVDataSource, error) {
	d, err := r.Get(id)
	if err != nil {
		return nil, err
	}
	return LoadCSV(d.FilePath, d.Symbol)
}

// ─── persistence ──────────────────────────────────────────────────────────────

func (r *Registry) catalogPath() string {
	return filepath.Join(r.catalogDir, catalogFile)
}

func (r *Registry) load() error {
	path := r.catalogPath()
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil // empty registry — not an error
	}
	if err != nil {
		return fmt.Errorf("dataset: open catalog %q: %w", path, err)
	}
	defer f.Close()

	var list []Dataset
	if err := json.NewDecoder(f).Decode(&list); err != nil {
		return fmt.Errorf("dataset: decode catalog: %w", err)
	}
	for _, d := range list {
		r.datasets[d.ID] = d
	}
	return nil
}

func (r *Registry) save() error {
	list := make([]Dataset, 0, len(r.datasets))
	for _, d := range r.datasets {
		list = append(list, d)
	}
	slices.SortFunc(list, func(a, b Dataset) int {
		return a.CreatedAt.Compare(b.CreatedAt)
	})

	tmp := r.catalogPath() + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("dataset: create catalog tmp: %w", err)
	}

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(list); err != nil {
		f.Close()
		os.Remove(tmp)
		return fmt.Errorf("dataset: encode catalog: %w", err)
	}
	f.Close()

	if err := os.Rename(tmp, r.catalogPath()); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("dataset: rename catalog: %w", err)
	}
	return nil
}

// ─── file utilities ───────────────────────────────────────────────────────────

// hashAndCount reads the file, computes its SHA-256 hex digest, and counts
// the number of non-header CSV rows.
func hashAndCount(filePath string) (hash string, count int, err error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()

	h := sha256.New()
	r := csv.NewReader(io.TeeReader(f, h))

	// skip header
	if _, err := r.Read(); err != nil {
		return "", 0, fmt.Errorf("read CSV header: %w", err)
	}

	for {
		_, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", 0, err
		}
		count++
	}

	return hex.EncodeToString(h.Sum(nil)), count, nil
}

// ─── CSVDataSource ────────────────────────────────────────────────────────────

// CSVDataSource implements strategies.HistoricalDataSource by serving candles
// from an in-memory slice loaded from a single-symbol OHLCV CSV file.
//
// Expected CSV header (case-insensitive): date,open,high,low,close,volume
// Date formats supported: 2006-01-02, 2006-01-02T15:04:05Z (RFC3339).
type CSVDataSource struct {
	symbol  string
	candles []strategies.Candle // sorted by Timestamp ascending
}

// LoadCSV reads the OHLCV CSV at filePath and returns a CSVDataSource for
// symbol.  All candles are loaded eagerly into memory.
func LoadCSV(filePath, symbol string) (*CSVDataSource, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("dataset.LoadCSV: %w", err)
	}
	defer f.Close()

	r := csv.NewReader(f)

	// Parse header to find column indices.
	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("dataset.LoadCSV: read header: %w", err)
	}
	colIdx := make(map[string]int, len(header))
	for i, h := range header {
		colIdx[strings.ToLower(strings.TrimSpace(h))] = i
	}
	idx := func(name string) (int, error) {
		i, ok := colIdx[name]
		if !ok {
			return 0, fmt.Errorf("CSV missing column %q", name)
		}
		return i, nil
	}

	dateCol, err := idx("date")
	if err != nil {
		return nil, fmt.Errorf("dataset.LoadCSV: %w", err)
	}
	openCol, err := idx("open")
	if err != nil {
		return nil, fmt.Errorf("dataset.LoadCSV: %w", err)
	}
	highCol, err := idx("high")
	if err != nil {
		return nil, fmt.Errorf("dataset.LoadCSV: %w", err)
	}
	lowCol, err := idx("low")
	if err != nil {
		return nil, fmt.Errorf("dataset.LoadCSV: %w", err)
	}
	closeCol, err := idx("close")
	if err != nil {
		return nil, fmt.Errorf("dataset.LoadCSV: %w", err)
	}
	volCol, err := idx("volume")
	if err != nil {
		return nil, fmt.Errorf("dataset.LoadCSV: %w", err)
	}

	symCol := -1
	if i, ok := colIdx["symbol"]; ok {
		symCol = i
	}

	dateFormats := []string{
		"2006-01-02",
		time.RFC3339,
		"2006-01-02 15:04:05",
	}

	parseDate := func(s string) (time.Time, error) {
		s = strings.TrimSpace(s)
		for _, layout := range dateFormats {
			if t, err := time.Parse(layout, s); err == nil {
				return t.UTC(), nil
			}
		}
		return time.Time{}, fmt.Errorf("unrecognised date format %q", s)
	}
	parseFloat := func(s string) (float64, error) {
		return strconv.ParseFloat(strings.TrimSpace(s), 64)
	}

	var candles []strategies.Candle
	lineNo := 1
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("dataset.LoadCSV: line %d: %w", lineNo+1, err)
		}
		lineNo++

		rowSymbol := symbol
		if symCol >= 0 && symCol < len(row) {
			rowSymbol = strings.TrimSpace(row[symCol])
		}

		ts, err := parseDate(row[dateCol])
		if err != nil {
			return nil, fmt.Errorf("dataset.LoadCSV: line %d date: %w", lineNo, err)
		}
		o, err := parseFloat(row[openCol])
		if err != nil {
			return nil, fmt.Errorf("dataset.LoadCSV: line %d open: %w", lineNo, err)
		}
		h2, err := parseFloat(row[highCol])
		if err != nil {
			return nil, fmt.Errorf("dataset.LoadCSV: line %d high: %w", lineNo, err)
		}
		l, err := parseFloat(row[lowCol])
		if err != nil {
			return nil, fmt.Errorf("dataset.LoadCSV: line %d low: %w", lineNo, err)
		}
		c, err := parseFloat(row[closeCol])
		if err != nil {
			return nil, fmt.Errorf("dataset.LoadCSV: line %d close: %w", lineNo, err)
		}
		v, err := strconv.ParseInt(strings.TrimSpace(row[volCol]), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("dataset.LoadCSV: line %d volume: %w", lineNo, err)
		}

		candles = append(candles, strategies.Candle{
			Symbol:    rowSymbol,
			Timestamp: ts,
			Open:      o,
			High:      h2,
			Low:       l,
			Close:     c,
			Volume:    v,
		})
	}

	return &CSVDataSource{symbol: symbol, candles: candles}, nil
}

// GetCandles returns all candles for symbol within [start, end] (inclusive).
// start and end are both inclusive; zero times mean "no bound".
func (ds *CSVDataSource) GetCandles(_ context.Context, symbol string, start, end time.Time) ([]strategies.Candle, error) {
	var out []strategies.Candle
	for _, c := range ds.candles {
		if c.Symbol != symbol {
			continue
		}
		if !start.IsZero() && c.Timestamp.Before(start) {
			continue
		}
		if !end.IsZero() && c.Timestamp.After(end) {
			continue
		}
		out = append(out, c)
	}
	return out, nil
}

// GetIndicators computes basic indicators for symbol at the closest candle
// timestamp on or before ts.  It derives:
//   - SMA20, SMA50, SMA200 from close prices
//   - ATR(14) from the raw candle data
//   - Volume, AvgVolume20
//
// MACD and BollingerBands are left zero-valued; extend this if strategies
// start requiring them.
func (ds *CSVDataSource) GetIndicators(_ context.Context, symbol string, ts time.Time) (strategies.AnalysisInput, error) {
	// Collect all candles for symbol up to and including ts.
	var history []strategies.Candle
	for _, c := range ds.candles {
		if c.Symbol != symbol {
			continue
		}
		if !ts.IsZero() && c.Timestamp.After(ts) {
			break
		}
		history = append(history, c)
	}

	if len(history) == 0 {
		return strategies.AnalysisInput{}, fmt.Errorf("dataset: no candles for %s at %s", symbol, ts)
	}

	last := history[len(history)-1]

	closes := make([]float64, len(history))
	for i, c := range history {
		closes[i] = c.Close
	}

	sma := func(n int) float64 {
		if len(closes) < n {
			return 0
		}
		slice := closes[len(closes)-n:]
		sum := 0.0
		for _, v := range slice {
			sum += v
		}
		return sum / float64(n)
	}

	// ATR(14)
	atr := computeATR(history, 14)

	// AvgVolume20
	var avgVol int64
	if len(history) >= 20 {
		var vsum int64
		for _, c := range history[len(history)-20:] {
			vsum += c.Volume
		}
		avgVol = vsum / 20
	}

	return strategies.AnalysisInput{
		Symbol:      symbol,
		Price:       last.Close,
		Timestamp:   last.Timestamp,
		SMA20:       sma(20),
		SMA50:       sma(50),
		SMA200:      sma(200),
		ATR:         atr,
		Volume:      last.Volume,
		AvgVolume20: avgVol,
		MarketTrend: marketTrend(closes),
	}, nil
}

// computeATR returns the Average True Range over the last n candles.
func computeATR(candles []strategies.Candle, n int) float64 {
	if len(candles) < 2 {
		return 0
	}
	start := 0
	if len(candles) > n+1 {
		start = len(candles) - n - 1
	}
	window := candles[start:]

	var trs []float64
	for i := 1; i < len(window); i++ {
		prev := window[i-1]
		cur := window[i]
		tr := math.Max(cur.High-cur.Low,
			math.Max(math.Abs(cur.High-prev.Close),
				math.Abs(cur.Low-prev.Close)))
		trs = append(trs, tr)
	}
	if len(trs) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range trs {
		sum += v
	}
	return sum / float64(len(trs))
}

// marketTrend returns a simple trend label from close prices.
func marketTrend(closes []float64) string {
	if len(closes) < 2 {
		return "neutral"
	}
	pct := (closes[len(closes)-1] - closes[0]) / closes[0]
	switch {
	case pct > 0.02:
		return "bullish"
	case pct < -0.02:
		return "bearish"
	default:
		return "neutral"
	}
}
