// Package portfoliorisk contains the deterministic, read-only Phase-09
// portfolio and risk decision boundary. It never creates execution objects.
package portfoliorisk

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	PortfolioContractVersion   = "jax.portfolio.snapshot/v1"
	PortfolioIdentityAlgorithm = "sha256-canonical-json/v1"
)

var currencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)

var (
	ErrInvalidSnapshot      = errors.New("portfolio snapshot is invalid")
	ErrUnknownMaterialState = errors.New("portfolio snapshot has unknown material state")
	ErrStalePortfolio       = errors.New("portfolio snapshot is stale")
	ErrSnapshotConflict     = errors.New("portfolio snapshot identity already contains different content")
	ErrSnapshotIdentity     = errors.New("portfolio snapshot identity does not match content")
)

// ObservedNumber deliberately separates an unknown number from the number
// zero. Unknown values must never be silently converted into risk-free values.
type ObservedNumber struct {
	Known  bool    `json:"known"`
	Value  float64 `json:"value"`
	Source string  `json:"source,omitempty"`
}

func KnownNumber(value float64, source string) ObservedNumber {
	return ObservedNumber{Known: true, Value: value, Source: strings.TrimSpace(source)}
}

func UnknownNumber(source string) ObservedNumber {
	return ObservedNumber{Known: false, Source: strings.TrimSpace(source)}
}

type Position struct {
	InstrumentID       string         `json:"instrument_id"`
	InstrumentResolved bool           `json:"instrument_resolved"`
	Currency           string         `json:"currency"`
	SignedQuantity     float64        `json:"signed_quantity"`
	Price              ObservedNumber `json:"price"`
	MarketValue        ObservedNumber `json:"market_value"`
	CostBasis          ObservedNumber `json:"cost_basis"`
	ValuationAsOf      time.Time      `json:"valuation_as_of"`
	PriceSource        string         `json:"price_source"`
	Provenance         []string       `json:"provenance"`
}

func (p Position) Direction() string {
	if p.SignedQuantity < 0 {
		return "SHORT"
	}
	return "LONG"
}

type PortfolioSnapshot struct {
	SnapshotID        string         `json:"snapshot_id"`
	ContractVersion   string         `json:"contract_version"`
	IdentityAlgorithm string         `json:"identity_algorithm"`
	AccountID         string         `json:"account_id"`
	AsOf              time.Time      `json:"as_of"`
	CapturedAt        time.Time      `json:"captured_at"`
	Provider          string         `json:"provider"`
	Currency          string         `json:"currency"`
	Synthetic         bool           `json:"synthetic"`
	Cash              ObservedNumber `json:"cash"`
	Equity            ObservedNumber `json:"equity"`
	Positions         []Position     `json:"positions"`
	Provenance        []string       `json:"provenance"`
	ValuationBasis    string         `json:"valuation_basis"`
}

// BuildSnapshot validates and content-addresses an immutable portfolio fact
// snapshot. The returned ID is independent of wall-clock time and slice order.
func BuildSnapshot(input PortfolioSnapshot) (PortfolioSnapshot, error) {
	claimedID := strings.TrimSpace(input.SnapshotID)
	input.SnapshotID = ""
	if input.ContractVersion == "" {
		input.ContractVersion = PortfolioContractVersion
	}
	if input.IdentityAlgorithm == "" {
		input.IdentityAlgorithm = PortfolioIdentityAlgorithm
	}
	if err := input.Validate(); err != nil {
		return PortfolioSnapshot{}, err
	}
	input.AsOf = input.AsOf.UTC()
	input.CapturedAt = input.CapturedAt.UTC()
	input.Currency = strings.ToUpper(strings.TrimSpace(input.Currency))
	input.Provider = strings.TrimSpace(input.Provider)
	input.AccountID = strings.TrimSpace(input.AccountID)
	input.ValuationBasis = strings.TrimSpace(input.ValuationBasis)
	input.Provenance = sortedStrings(input.Provenance)
	input.Positions = append([]Position(nil), input.Positions...)
	for i := range input.Positions {
		input.Positions[i].InstrumentID = strings.TrimSpace(input.Positions[i].InstrumentID)
		input.Positions[i].Currency = strings.ToUpper(strings.TrimSpace(input.Positions[i].Currency))
		input.Positions[i].PriceSource = strings.TrimSpace(input.Positions[i].PriceSource)
		input.Positions[i].Provenance = sortedStrings(input.Positions[i].Provenance)
		input.Positions[i].ValuationAsOf = input.Positions[i].ValuationAsOf.UTC()
	}
	sort.Slice(input.Positions, func(i, j int) bool {
		if input.Positions[i].InstrumentID != input.Positions[j].InstrumentID {
			return input.Positions[i].InstrumentID < input.Positions[j].InstrumentID
		}
		return input.Positions[i].SignedQuantity < input.Positions[j].SignedQuantity
	})
	input.SnapshotID = snapshotIdentity(input)
	if claimedID != "" && claimedID != input.SnapshotID {
		return PortfolioSnapshot{}, fmt.Errorf("%w: claimed %q calculated %q", ErrSnapshotIdentity, claimedID, input.SnapshotID)
	}
	return input, nil
}

func (s PortfolioSnapshot) Validate() error {
	if s.ContractVersion != "" && s.ContractVersion != PortfolioContractVersion {
		return fmt.Errorf("%w: unsupported contract version %q", ErrInvalidSnapshot, s.ContractVersion)
	}
	if s.IdentityAlgorithm != "" && s.IdentityAlgorithm != PortfolioIdentityAlgorithm {
		return fmt.Errorf("%w: unsupported identity algorithm %q", ErrInvalidSnapshot, s.IdentityAlgorithm)
	}
	if strings.TrimSpace(s.AccountID) == "" || strings.TrimSpace(s.Provider) == "" {
		return fmt.Errorf("%w: account_id and provider are required", ErrInvalidSnapshot)
	}
	if s.AsOf.IsZero() || s.CapturedAt.IsZero() || s.CapturedAt.Before(s.AsOf) {
		return fmt.Errorf("%w: as_of and captured_at are required and capture cannot precede as_of", ErrInvalidSnapshot)
	}
	if !currencyPattern.MatchString(strings.ToUpper(strings.TrimSpace(s.Currency))) {
		return fmt.Errorf("%w: currency must be a three-letter uppercase code", ErrInvalidSnapshot)
	}
	if err := validateObserved(s.Cash, "cash"); err != nil {
		return err
	}
	if err := validateObserved(s.Equity, "equity"); err != nil {
		return err
	}
	if s.Cash.Known && s.Cash.Value < 0 {
		return fmt.Errorf("%w: known cash cannot be negative in Phase 09", ErrInvalidSnapshot)
	}
	if len(s.Positions) == 0 {
		return fmt.Errorf("%w: at least one position record is required; use an empty known snapshot explicitly only at a later policy boundary", ErrInvalidSnapshot)
	}
	for i, p := range s.Positions {
		if strings.TrimSpace(p.InstrumentID) == "" || !finiteNonZero(p.SignedQuantity) {
			return fmt.Errorf("%w: position %d requires a finite non-zero signed quantity and instrument", ErrInvalidSnapshot, i)
		}
		if !currencyPattern.MatchString(strings.ToUpper(strings.TrimSpace(p.Currency))) || p.Currency != "" && strings.ToUpper(strings.TrimSpace(p.Currency)) != strings.ToUpper(strings.TrimSpace(s.Currency)) {
			return fmt.Errorf("%w: position %d currency must match portfolio currency", ErrInvalidSnapshot, i)
		}
		if err := validateObserved(p.Price, fmt.Sprintf("position %d price", i)); err != nil {
			return err
		}
		if p.Price.Known && p.Price.Value <= 0 {
			return fmt.Errorf("%w: position %d known price must be positive", ErrInvalidSnapshot, i)
		}
		if err := validateObserved(p.MarketValue, fmt.Sprintf("position %d market value", i)); err != nil {
			return err
		}
		if err := validateObserved(p.CostBasis, fmt.Sprintf("position %d cost basis", i)); err != nil {
			return err
		}
		if !p.ValuationAsOf.IsZero() && p.ValuationAsOf.After(s.AsOf) {
			return fmt.Errorf("%w: position %d valuation is after portfolio as_of", ErrInvalidSnapshot, i)
		}
	}
	return nil
}

func validateObserved(v ObservedNumber, field string) error {
	if v.Known && !finite(v.Value) {
		return fmt.Errorf("%w: %s is non-finite", ErrInvalidSnapshot, field)
	}
	return nil
}

func finite(value float64) bool        { return !math.IsNaN(value) && !math.IsInf(value, 0) }
func finiteNonZero(value float64) bool { return finite(value) && value != 0 }

type FreshnessStatus string

const (
	FreshnessFresh   FreshnessStatus = "FRESH"
	FreshnessStale   FreshnessStatus = "STALE"
	FreshnessUnknown FreshnessStatus = "UNKNOWN"
)

type FreshnessResult struct {
	Status FreshnessStatus `json:"status"`
	Reason string          `json:"reason"`
}

// AssessFreshness applies one explicit age rule to the account timestamp and
// every known valuation timestamp. Missing valuation, unresolved instruments,
// and unsupported base currencies remain UNKNOWN rather than becoming zero.
func (s PortfolioSnapshot) AssessFreshness(now time.Time, maxAge time.Duration) FreshnessResult {
	if now.IsZero() || maxAge <= 0 || s.AsOf.IsZero() || s.CapturedAt.IsZero() {
		return FreshnessResult{Status: FreshnessUnknown, Reason: "freshness inputs are incomplete"}
	}
	now = now.UTC()
	if !SupportedCurrency(s.Currency) {
		return FreshnessResult{Status: FreshnessUnknown, Reason: "unsupported portfolio currency"}
	}
	if !s.Equity.Known || !s.Cash.Known {
		return FreshnessResult{Status: FreshnessUnknown, Reason: "cash and equity must be known for risk evaluation"}
	}
	for i, p := range s.Positions {
		if !p.InstrumentResolved {
			return FreshnessResult{Status: FreshnessUnknown, Reason: fmt.Sprintf("position %d instrument is unresolved", i)}
		}
		if !p.Price.Known || !p.MarketValue.Known || p.ValuationAsOf.IsZero() {
			return FreshnessResult{Status: FreshnessUnknown, Reason: fmt.Sprintf("position %d valuation is unknown", i)}
		}
		if now.Sub(p.ValuationAsOf.UTC()) > maxAge {
			return FreshnessResult{Status: FreshnessStale, Reason: fmt.Sprintf("position %d valuation exceeds freshness window", i)}
		}
	}
	if now.Sub(s.AsOf.UTC()) > maxAge || now.Sub(s.CapturedAt.UTC()) > maxAge {
		return FreshnessResult{Status: FreshnessStale, Reason: "account snapshot exceeds freshness window"}
	}
	return FreshnessResult{Status: FreshnessFresh, Reason: "all material account and valuation facts are fresh"}
}

func SupportedCurrency(currency string) bool {
	switch strings.ToUpper(strings.TrimSpace(currency)) {
	case "USD", "GBP", "EUR", "JPY", "CAD", "AUD", "CHF":
		return true
	default:
		return false
	}
}

func snapshotIdentity(s PortfolioSnapshot) string {
	b, _ := json.Marshal(s)
	digest := sha256.Sum256(b)
	return "pf_" + hex.EncodeToString(digest[:])
}

func sortedStrings(values []string) []string {
	copyValues := append([]string(nil), values...)
	sort.Strings(copyValues)
	return copyValues
}

// SnapshotStore is intentionally append-only. It persists observed facts, not
// derived risk outcomes or execution state.
type SnapshotStore interface {
	Save(context.Context, PortfolioSnapshot) error
	Get(context.Context, string) (PortfolioSnapshot, error)
}

type MemorySnapshotStore struct {
	mu    sync.RWMutex
	items map[string]PortfolioSnapshot
}

func NewMemorySnapshotStore() *MemorySnapshotStore {
	return &MemorySnapshotStore{items: make(map[string]PortfolioSnapshot)}
}

func (s *MemorySnapshotStore) Save(_ context.Context, snapshot PortfolioSnapshot) error {
	canonical, err := BuildSnapshot(snapshot)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.items[canonical.SnapshotID]; ok {
		if !snapshotsEqual(existing, canonical) {
			return ErrSnapshotConflict
		}
		return nil
	}
	s.items[canonical.SnapshotID] = canonical
	return nil
}

func (s *MemorySnapshotStore) Get(_ context.Context, id string) (PortfolioSnapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.items[strings.TrimSpace(id)]
	if !ok {
		return PortfolioSnapshot{}, sql.ErrNoRows
	}
	return item, nil
}

func snapshotsEqual(left, right PortfolioSnapshot) bool {
	left.SnapshotID, right.SnapshotID = "", ""
	a, _ := json.Marshal(left)
	b, _ := json.Marshal(right)
	return string(a) == string(b)
}
