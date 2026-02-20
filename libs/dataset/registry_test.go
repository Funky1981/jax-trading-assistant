package dataset_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"jax-trading-assistant/libs/dataset"
)

// ─── helpers ──────────────────────────────────────────────────────────────────

func writeTempCSV(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writeTempCSV: %v", err)
	}
	return path
}

const sampleCSV = `date,open,high,low,close,volume
2024-01-02,150.00,155.00,148.00,153.00,1000000
2024-01-03,153.00,158.00,151.00,156.00,1200000
2024-01-04,156.00,160.00,154.00,157.00,900000
2024-01-05,157.00,161.00,155.00,159.00,1100000
2024-01-08,159.00,163.00,157.00,162.00,1050000
`

// ─── Registry tests ───────────────────────────────────────────────────────────

func TestOpenCreatesDir(t *testing.T) {
	dir := t.TempDir()
	catalogDir := filepath.Join(dir, "new", "registry")
	_, err := dataset.Open(catalogDir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := os.Stat(catalogDir); err != nil {
		t.Fatalf("catalog dir not created: %v", err)
	}
}

func TestRegisterAndGet(t *testing.T) {
	dir := t.TempDir()
	csvPath := writeTempCSV(t, dir, "aapl.csv", sampleCSV)

	reg, err := dataset.Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	d, err := reg.Register(dataset.Dataset{
		Name:     "AAPL_2024_test",
		Symbol:   "AAPL",
		FilePath: csvPath,
		Source:   "csv",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	if d.ID == "" {
		t.Error("expected non-empty ID")
	}
	if d.Hash == "" {
		t.Error("expected non-empty Hash")
	}
	if d.RecordCount != 5 {
		t.Errorf("RecordCount: got %d, want 5", d.RecordCount)
	}
	if d.SchemaVer == "" {
		t.Error("expected non-empty SchemaVer")
	}

	got, err := reg.Get(d.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != d.Name {
		t.Errorf("Name mismatch: got %q want %q", got.Name, d.Name)
	}
}

func TestRegisterDuplicateNameReturnsError(t *testing.T) {
	dir := t.TempDir()
	csvPath := writeTempCSV(t, dir, "spy.csv", sampleCSV)

	reg, _ := dataset.Open(dir)

	if _, err := reg.Register(dataset.Dataset{Name: "SPY_test", Symbol: "SPY", FilePath: csvPath}); err != nil {
		t.Fatalf("first Register: %v", err)
	}
	if _, err := reg.Register(dataset.Dataset{Name: "SPY_test", Symbol: "SPY", FilePath: csvPath}); err == nil {
		t.Fatal("expected error for duplicate name, got nil")
	}
}

func TestRegisterMissingFileReturnsError(t *testing.T) {
	dir := t.TempDir()
	reg, _ := dataset.Open(dir)

	_, err := reg.Register(dataset.Dataset{Name: "X", Symbol: "X", FilePath: "/nonexistent.csv"})
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestRegisterMissingNameReturnsError(t *testing.T) {
	dir := t.TempDir()
	reg, _ := dataset.Open(dir)

	_, err := reg.Register(dataset.Dataset{Symbol: "X", FilePath: "/any.csv"})
	if err == nil {
		t.Fatal("expected error for empty Name, got nil")
	}
}

func TestGetByName(t *testing.T) {
	dir := t.TempDir()
	csvPath := writeTempCSV(t, dir, "msft.csv", sampleCSV)
	reg, _ := dataset.Open(dir)

	want, _ := reg.Register(dataset.Dataset{Name: "MSFT_Q1", Symbol: "MSFT", FilePath: csvPath})

	got, err := reg.GetByName("MSFT_Q1")
	if err != nil {
		t.Fatalf("GetByName: %v", err)
	}
	if got.ID != want.ID {
		t.Errorf("ID mismatch: got %s want %s", got.ID, want.ID)
	}
}

func TestList(t *testing.T) {
	dir := t.TempDir()
	csv1 := writeTempCSV(t, dir, "a.csv", sampleCSV)
	csv2 := writeTempCSV(t, dir, "b.csv", sampleCSV)
	reg, _ := dataset.Open(dir)

	reg.Register(dataset.Dataset{Name: "A", Symbol: "A", FilePath: csv1}) //nolint:errcheck
	reg.Register(dataset.Dataset{Name: "B", Symbol: "B", FilePath: csv2}) //nolint:errcheck

	list := reg.List()
	if len(list) != 2 {
		t.Fatalf("List: got %d, want 2", len(list))
	}
}

func TestRemove(t *testing.T) {
	dir := t.TempDir()
	csvPath := writeTempCSV(t, dir, "c.csv", sampleCSV)
	reg, _ := dataset.Open(dir)

	d, _ := reg.Register(dataset.Dataset{Name: "C", Symbol: "C", FilePath: csvPath})

	if err := reg.Remove(d.ID); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, err := reg.Get(d.ID); err == nil {
		t.Fatal("expected error after Remove, got nil")
	}
}

func TestVerifyHashDetectsChange(t *testing.T) {
	dir := t.TempDir()
	csvPath := writeTempCSV(t, dir, "chg.csv", sampleCSV)
	reg, _ := dataset.Open(dir)

	d, _ := reg.Register(dataset.Dataset{Name: "CHG", Symbol: "CHG", FilePath: csvPath})

	// Initially intact.
	if err := reg.VerifyHash(d.ID); err != nil {
		t.Fatalf("VerifyHash (intact): %v", err)
	}

	// Mutate the file.
	os.WriteFile(csvPath, []byte(sampleCSV+"2024-01-09,163,167,161,165,900000\n"), 0o644) //nolint:errcheck

	if err := reg.VerifyHash(d.ID); err == nil {
		t.Fatal("expected hash mismatch error, got nil")
	}
}

// TestPersistence verifies the catalog survives reopen.
func TestPersistence(t *testing.T) {
	dir := t.TempDir()
	csvPath := writeTempCSV(t, dir, "persist.csv", sampleCSV)

	reg1, _ := dataset.Open(dir)
	d, _ := reg1.Register(dataset.Dataset{Name: "PERSIST", Symbol: "P", FilePath: csvPath})

	// Reopen from the same directory.
	reg2, err := dataset.Open(dir)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	got, err := reg2.Get(d.ID)
	if err != nil {
		t.Fatalf("Get after reopen: %v", err)
	}
	if got.Hash != d.Hash {
		t.Errorf("Hash changed across reopen: %s vs %s", got.Hash, d.Hash)
	}
}

// ─── CSVDataSource tests ──────────────────────────────────────────────────────

func TestLoadCSVGetCandles(t *testing.T) {
	dir := t.TempDir()
	csvPath := writeTempCSV(t, dir, "candles.csv", sampleCSV)

	ds, err := dataset.LoadCSV(csvPath, "AAPL")
	if err != nil {
		t.Fatalf("LoadCSV: %v", err)
	}

	ctx := context.Background()

	// All candles.
	candles, err := ds.GetCandles(ctx, "AAPL", time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("GetCandles: %v", err)
	}
	if len(candles) != 5 {
		t.Fatalf("GetCandles all: got %d, want 5", len(candles))
	}

	// Date-filtered.
	start := time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 4, 23, 59, 59, 0, time.UTC)
	sub, err := ds.GetCandles(ctx, "AAPL", start, end)
	if err != nil {
		t.Fatalf("GetCandles filtered: %v", err)
	}
	if len(sub) != 2 {
		t.Errorf("GetCandles filtered: got %d, want 2", len(sub))
	}

	// Wrong symbol.
	none, err := ds.GetCandles(ctx, "GOOG", time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("GetCandles wrong symbol: %v", err)
	}
	if len(none) != 0 {
		t.Errorf("expected 0 candles for GOOG, got %d", len(none))
	}
}

func TestGetIndicatorsReturnsValues(t *testing.T) {
	dir := t.TempDir()
	csvPath := writeTempCSV(t, dir, "ind.csv", sampleCSV)

	ds, err := dataset.LoadCSV(csvPath, "AAPL")
	if err != nil {
		t.Fatalf("LoadCSV: %v", err)
	}

	ctx := context.Background()
	ts := time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC)

	input, err := ds.GetIndicators(ctx, "AAPL", ts)
	if err != nil {
		t.Fatalf("GetIndicators: %v", err)
	}

	if input.Symbol != "AAPL" {
		t.Errorf("Symbol: got %q want %q", input.Symbol, "AAPL")
	}
	if input.Price <= 0 {
		t.Errorf("Price should be positive, got %f", input.Price)
	}
	if input.ATR <= 0 {
		t.Errorf("ATR should be positive, got %f", input.ATR)
	}
	if input.Volume <= 0 {
		t.Errorf("Volume should be positive, got %d", input.Volume)
	}
}

func TestGetIndicatorsNoDataReturnsError(t *testing.T) {
	dir := t.TempDir()
	csvPath := writeTempCSV(t, dir, "empty.csv", "date,open,high,low,close,volume\n")

	ds, err := dataset.LoadCSV(csvPath, "AAPL")
	if err != nil {
		t.Fatalf("LoadCSV: %v", err)
	}

	_, err = ds.GetIndicators(context.Background(), "AAPL", time.Now())
	if err == nil {
		t.Fatal("expected error for empty dataset, got nil")
	}
}

func TestLoadCSVMissingFile(t *testing.T) {
	_, err := dataset.LoadCSV("/no/such/file.csv", "X")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestLoadCSVBadHeader(t *testing.T) {
	dir := t.TempDir()
	csvPath := writeTempCSV(t, dir, "bad.csv", "ts,price\n2024-01-01,100\n")
	_, err := dataset.LoadCSV(csvPath, "X")
	if err == nil {
		t.Fatal("expected error for missing columns, got nil")
	}
}
