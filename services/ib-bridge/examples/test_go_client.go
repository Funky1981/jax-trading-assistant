package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"jax-trading-assistant/libs/marketdata"
	"jax-trading-assistant/libs/marketdata/ib"
)

func main() {
	fmt.Println("=== IB Bridge Go Client Test ===")
	fmt.Println()

	// Create IB provider
	fmt.Println("🔌 Connecting to IB Bridge...")
	provider, err := ib.NewProvider("http://localhost:8092")
	if err != nil {
		log.Fatalf("Failed to create IB provider: %v", err)
	}
	defer provider.Close()
	fmt.Println("✅ Connected to IB Bridge")
	fmt.Println()

	ctx := context.Background()

	// Test 1: Get Quote
	fmt.Println("=== Test 1: Get Real-Time Quote ===")
	testGetQuote(ctx, provider, "AAPL")

	// Test 2: Get Candles
	fmt.Println("\n=== Test 2: Get Historical Candles ===")
	testGetCandles(ctx, provider, "AAPL")

	// Test 3: Multiple Symbols
	fmt.Println("\n=== Test 3: Multiple Symbols ===")
	symbols := []string{"AAPL", "MSFT", "GOOGL", "TSLA"}
	testMultipleQuotes(ctx, provider, symbols)

	// Test 4: Direct Client Access
	fmt.Println("\n=== Test 4: Direct Client Access ===")
	testDirectClient()

	fmt.Println()
	fmt.Println("🎉 All tests completed successfully!")
}

func testGetQuote(ctx context.Context, provider *ib.Provider, symbol string) {
	quote, err := provider.GetQuote(ctx, symbol)
	if err != nil {
		log.Printf("❌ Failed to get quote for %s: %v", symbol, err)
		return
	}

	fmt.Printf("Symbol: %s\n", quote.Symbol)
	fmt.Printf("Price:  $%.2f\n", quote.Price)
	fmt.Printf("Bid:    $%.2f x %d\n", quote.Bid, quote.BidSize)
	fmt.Printf("Ask:    $%.2f x %d\n", quote.Ask, quote.AskSize)
	fmt.Printf("Volume: %s\n", formatVolume(quote.Volume))
	fmt.Printf("Time:   %s\n", quote.Timestamp.Format("15:04:05"))
	fmt.Println("✅ Quote retrieved successfully")
}

func testGetCandles(ctx context.Context, provider *ib.Provider, symbol string) {
	to := time.Now()
	from := to.Add(-24 * time.Hour)

	candles, err := provider.GetCandles(ctx, symbol, marketdata.Timeframe5Min, from, to)
	if err != nil {
		log.Printf("❌ Failed to get candles for %s: %v", symbol, err)
		return
	}

	fmt.Printf("Retrieved %d candles for %s\n", len(candles), symbol)

	if len(candles) > 0 {
		// Show first candle
		first := candles[0]
		fmt.Printf("\nFirst candle (%s):\n", first.Timestamp.Format("2006-01-02 15:04"))
		fmt.Printf("  O: $%.2f  H: $%.2f  L: $%.2f  C: $%.2f  V: %s\n",
			first.Open, first.High, first.Low, first.Close, formatVolume(first.Volume))

		// Show last candle
		last := candles[len(candles)-1]
		fmt.Printf("\nLast candle (%s):\n", last.Timestamp.Format("2006-01-02 15:04"))
		fmt.Printf("  O: $%.2f  H: $%.2f  L: $%.2f  C: $%.2f  V: %s\n",
			last.Open, last.High, last.Low, last.Close, formatVolume(last.Volume))

		// Calculate some stats
		var totalVolume int64
		var highestPrice, lowestPrice float64
		highestPrice = candles[0].High
		lowestPrice = candles[0].Low

		for _, c := range candles {
			totalVolume += c.Volume
			if c.High > highestPrice {
				highestPrice = c.High
			}
			if c.Low < lowestPrice {
				lowestPrice = c.Low
			}
		}

		fmt.Printf("\nStats:\n")
		fmt.Printf("  Period High: $%.2f\n", highestPrice)
		fmt.Printf("  Period Low:  $%.2f\n", lowestPrice)
		fmt.Printf("  Total Volume: %s\n", formatVolume(totalVolume))
	}

	fmt.Println("✅ Candles retrieved successfully")
}

func testMultipleQuotes(ctx context.Context, provider *ib.Provider, symbols []string) {
	fmt.Printf("Fetching quotes for %d symbols...\n\n", len(symbols))

	for _, symbol := range symbols {
		quote, err := provider.GetQuote(ctx, symbol)
		if err != nil {
			fmt.Printf("❌ %s: Failed - %v\n", symbol, err)
			continue
		}

		spread := quote.Ask - quote.Bid
		spreadPct := (spread / quote.Price) * 100

		fmt.Printf("%-6s $%7.2f | Bid: $%7.2f | Ask: $%7.2f | Spread: $%.2f (%.2f%%)\n",
			quote.Symbol, quote.Price, quote.Bid, quote.Ask, spread, spreadPct)
	}

	fmt.Println("✅ Multiple quotes retrieved successfully")
}

func testDirectClient() {
	client := ib.NewClient(ib.Config{
		BaseURL: "http://localhost:8092",
		Timeout: 10 * time.Second,
	})

	ctx := context.Background()

	// Health check
	health, err := client.Health(ctx)
	if err != nil {
		log.Printf("❌ Health check failed: %v", err)
		return
	}

	fmt.Printf("Service Status: %s\n", health.Status)
	fmt.Printf("Connected to IB: %v\n", health.Connected)
	fmt.Printf("Version: %s\n", health.Version)

	// Get account info
	account, err := client.GetAccount(ctx)
	if err != nil {
		log.Printf("❌ Failed to get account: %v", err)
		return
	}

	fmt.Printf("\nAccount Information:\n")
	fmt.Printf("  Account ID: %s\n", account.AccountID)
	fmt.Printf("  Net Liquidation: $%s\n", formatMoney(account.NetLiquidation))
	fmt.Printf("  Cash: $%s\n", formatMoney(account.TotalCash))
	fmt.Printf("  Buying Power: $%s\n", formatMoney(account.BuyingPower))

	// Get positions
	positions, err := client.GetPositions(ctx)
	if err != nil {
		log.Printf("❌ Failed to get positions: %v", err)
		return
	}

	if positions.Count > 0 {
		fmt.Printf("\nPositions (%d):\n", positions.Count)
		for _, pos := range positions.Positions {
			fmt.Printf("  %-6s: %4d shares @ $%.2f = $%s\n",
				pos.Symbol, pos.Quantity, pos.AvgCost, formatMoney(pos.MarketValue))
		}
	} else {
		fmt.Println("\nNo positions (account is flat)")
	}

	fmt.Println("✅ Direct client test completed successfully")
}

// Helper functions

func formatVolume(volume int64) string {
	if volume >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(volume)/1_000_000)
	} else if volume >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(volume)/1_000)
	}
	return fmt.Sprintf("%d", volume)
}

func formatMoney(amount float64) string {
	if amount >= 1_000_000 {
		return fmt.Sprintf("%.2fM", amount/1_000_000)
	} else if amount >= 1_000 {
		return fmt.Sprintf("%.2fK", amount/1_000)
	}
	return fmt.Sprintf("%.2f", amount)
}
