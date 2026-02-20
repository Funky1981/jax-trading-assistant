package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ShadowValidator compares trader runtime decisions against legacy executor
type ShadowValidator struct {
	productionDB *pgxpool.Pool
	shadowDB     *pgxpool.Pool
}

type TradeDecision struct {
	SignalID   string    `json:"signal_id"`
	Symbol     string    `json:"symbol"`
	Action     string    `json:"action"`
	Quantity   float64   `json:"quantity"`
	Price      float64   `json:"price"`
	ArtifactID string    `json:"artifact_id,omitempty"`
	ExecutedAt time.Time `json:"executed_at"`
	OrderType  string    `json:"order_type"`
}

type Discrepancy struct {
	SignalID        string      `json:"signal_id"`
	Issue           string      `json:"issue"`
	ProductionValue interface{} `json:"production_value,omitempty"`
	ShadowValue     interface{} `json:"shadow_value,omitempty"`
}

func main() {
	log.Println("🔍 Shadow Mode Validation - ADR-0012 Phase 4.3")
	log.Println("Comparing trader runtime decisions against production baseline")
	log.Println("")

	// Connect to production database
	productionURL := os.Getenv("PRODUCTION_DATABASE_URL")
	if productionURL == "" {
		productionURL = "postgresql://jax:password@localhost:5433/jax"
	}

	productionDB, err := pgxpool.New(context.Background(), productionURL)
	if err != nil {
		log.Fatalf("❌ Failed to connect to production DB: %v", err)
	}
	defer productionDB.Close()

	// Connect to shadow database
	shadowURL := os.Getenv("SHADOW_DATABASE_URL")
	if shadowURL == "" {
		shadowURL = "postgresql://jax:password@localhost:5433/jax_shadow"
	}

	shadowDB, err := pgxpool.New(context.Background(), shadowURL)
	if err != nil {
		log.Fatalf("❌ Failed to connect to shadow DB: %v", err)
	}
	defer shadowDB.Close()

	validator := &ShadowValidator{
		productionDB: productionDB,
		shadowDB:     shadowDB,
	}

	// Get time window for comparison
	windowHours := 24
	if hours := os.Getenv("COMPARISON_WINDOW_HOURS"); hours != "" {
		fmt.Sscanf(hours, "%d", &windowHours)
	}

	log.Printf("Comparing last %d hours of decisions\n\n", windowHours)

	// Run comparison
	discrepancies, err := validator.CompareDecisions(context.Background(), time.Duration(windowHours)*time.Hour)
	if err != nil {
		log.Fatalf("❌ Comparison failed: %v", err)
	}

	// Report results
	if len(discrepancies) > 0 {
		log.Printf("❌ Found %d discrepancies\n\n", len(discrepancies))

		for i, d := range discrepancies {
			log.Printf("Discrepancy %d:\n", i+1)
			log.Printf("  Signal ID: %s\n", d.SignalID)
			log.Printf("  Issue: %s\n", d.Issue)
			if d.ProductionValue != nil {
				log.Printf("  Production: %v\n", d.ProductionValue)
			}
			if d.ShadowValue != nil {
				log.Printf("  Shadow: %v\n", d.ShadowValue)
			}
			log.Println()
		}

		// Save report
		reportFile := fmt.Sprintf("shadow-validation-report-%s.json", time.Now().Format("2006-01-02-15-04-05"))
		if err := saveReport(discrepancies, reportFile); err != nil {
			log.Printf("⚠️  Failed to save report: %v", err)
		} else {
			log.Printf("📄 Report saved to: %s\n", reportFile)
		}

		os.Exit(1)
	}

	log.Println("✅ Shadow validation PASSED - All decisions match!")
	log.Println("   Safe to proceed with production cutover")
	os.Exit(0)
}

func (v *ShadowValidator) CompareDecisions(ctx context.Context, window time.Duration) ([]Discrepancy, error) {
	since := time.Now().Add(-window)

	// Query production trades (legacy executor)
	productionTrades, err := v.queryTrades(ctx, v.productionDB, since, "production")
	if err != nil {
		return nil, fmt.Errorf("failed to query production: %w", err)
	}

	// Query shadow trades (new trader runtime)
	shadowTrades, err := v.queryTrades(ctx, v.shadowDB, since, "shadow")
	if err != nil {
		return nil, fmt.Errorf("failed to query shadow: %w", err)
	}

	log.Printf("📊 Production trades: %d\n", len(productionTrades))
	log.Printf("📊 Shadow trades: %d\n\n", len(shadowTrades))

	// Compare
	var discrepancies []Discrepancy

	// Check for trades in production but missing in shadow
	for signalID, prodTrade := range productionTrades {
		shadowTrade, exists := shadowTrades[signalID]
		if !exists {
			discrepancies = append(discrepancies, Discrepancy{
				SignalID:        signalID,
				Issue:           "Trade executed in production but NOT in shadow",
				ProductionValue: prodTrade,
			})
			continue
		}

		// Compare position sizes (allow 0.01 rounding difference)
		if math.Abs(prodTrade.Quantity-shadowTrade.Quantity) > 0.01 {
			discrepancies = append(discrepancies, Discrepancy{
				SignalID:        signalID,
				Issue:           "Position size mismatch",
				ProductionValue: prodTrade.Quantity,
				ShadowValue:     shadowTrade.Quantity,
			})
		}

		// Compare actions
		if prodTrade.Action != shadowTrade.Action {
			discrepancies = append(discrepancies, Discrepancy{
				SignalID:        signalID,
				Issue:           "Action mismatch (BUY vs SELL)",
				ProductionValue: prodTrade.Action,
				ShadowValue:     shadowTrade.Action,
			})
		}

		// Compare order types
		if prodTrade.OrderType != shadowTrade.OrderType {
			discrepancies = append(discrepancies, Discrepancy{
				SignalID:        signalID,
				Issue:           "Order type mismatch",
				ProductionValue: prodTrade.OrderType,
				ShadowValue:     shadowTrade.OrderType,
			})
		}

		// Verify shadow has artifact tracking
		if shadowTrade.ArtifactID == "" {
			discrepancies = append(discrepancies, Discrepancy{
				SignalID: signalID,
				Issue:    "Shadow trade missing artifact_id (audit trail broken)",
			})
		}
	}

	// Check for trades in shadow but missing in production
	for signalID, shadowTrade := range shadowTrades {
		if _, exists := productionTrades[signalID]; !exists {
			discrepancies = append(discrepancies, Discrepancy{
				SignalID:    signalID,
				Issue:       "Trade executed in shadow but NOT in production",
				ShadowValue: shadowTrade,
			})
		}
	}

	return discrepancies, nil
}

func (v *ShadowValidator) queryTrades(ctx context.Context, db *pgxpool.Pool, since time.Time, label string) (map[string]*TradeDecision, error) {
	query := `
		SELECT signal_id, symbol, action, quantity, price, 
		       COALESCE(artifact_id, ''), executed_at, 
		       COALESCE(order_type, 'UNKNOWN')
		FROM trades
		WHERE executed_at >= $1
		ORDER BY executed_at
	`

	rows, err := db.Query(ctx, query, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	trades := make(map[string]*TradeDecision)
	for rows.Next() {
		var trade TradeDecision
		var executedAt time.Time

		err := rows.Scan(
			&trade.SignalID,
			&trade.Symbol,
			&trade.Action,
			&trade.Quantity,
			&trade.Price,
			&trade.ArtifactID,
			&executedAt,
			&trade.OrderType,
		)
		if err != nil {
			return nil, err
		}

		trade.ExecutedAt = executedAt
		trades[trade.SignalID] = &trade
	}

	return trades, rows.Err()
}

func saveReport(discrepancies []Discrepancy, filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")

	report := map[string]interface{}{
		"timestamp":         time.Now(),
		"discrepancy_count": len(discrepancies),
		"discrepancies":     discrepancies,
	}

	return enc.Encode(report)
}
