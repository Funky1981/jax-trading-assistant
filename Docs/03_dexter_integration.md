
# 03 – Dexter Integration (Research Engine)

Goal: integrate **Dexter** as a first‑class UTCP provider so that Jax Core and
the Agent0‑style lab can ask it financial research questions.

Dexter remains its own service. Jax only talks to it as a **tool provider**.

---

## 1. Dexter Service Expectations

The existing Dexter repo is a TypeScript/Bun project that already knows how to:

- Fetch financial data.
- Run multi‑step research agents.
- Produce structured analysis.

For this integration we assume Dexter can:

- Run as a long‑lived service (e.g. `bun run dev` or similar).  
- Expose either:
  - HTTP endpoints, **or**
  - A CLI that we wrap behind a small HTTP adapter.

We target HTTP for simplicity and portability.

---

## 2. Dexter HTTP Contract

Create a small HTTP layer for Dexter with at least two endpoints:

1. `POST /research/company`
2. `POST /research/compare`

### 2.1 Endpoint: POST /research/company

**Request JSON:**

```json
{
  "ticker": "AAPL",
  "questions": [
    "Summarise the last 4 quarters of earnings.",
    "Highlight the main risks and opportunities.",
    "Describe the main revenue drivers."
  ]
}
```

**Response JSON:**

```json
{
  "ticker": "AAPL",
  "summary": "High revenue growth with stable margins...",
  "key_points": [
    "Revenue has grown YoY for 4 consecutive quarters.",
    "Margins stable but capex is increasing.",
    "Significant dependency on product and service ecosystem."
  ],
  "metrics": {
    "pe_ratio": 28.5,
    "eps_growth_1y": 0.14,
    "debt_to_equity": 0.7
  },
  "raw_markdown": "## AAPL Research\n\n- Bullet 1\n- Bullet 2\n"
}
```

### 2.2 Endpoint: POST /research/compare

**Request JSON:**

```json
{
  "tickers": ["AAPL", "MSFT", "GOOGL"],
  "focus": "growth_vs_margins"
}
```

**Response JSON:**

```json
{
  "comparison_axis": "growth_vs_margins",
  "items": [
    {
      "ticker": "AAPL",
      "thesis": "Moderate growth, strong margins.",
      "notes": ["..."]
    },
    {
      "ticker": "MSFT",
      "thesis": "Cloud‑driven growth with high margins.",
      "notes": ["..."]
    }
  ]
}
```

The exact wording can vary; the important part is the **shape**.

---

## 3. UTCP Dexter Provider

In Jax, declare a provider in `config/providers.json`:

```json
{
  "id": "dexter",
  "transport": "http",
  "endpoint": "http://dexter:3000/tools"
}
```

Then expose the following tools (the HTTP layer behind `/tools` may proxy to
the underlying Dexter endpoints):

- `dexter.research_company`
- `dexter.compare_companies`

### 3.1 dexter.research_company Tool

Input shape:

```json
{
  "ticker": "AAPL",
  "questions": [
    "Summarise last 4 quarters of earnings.",
    "Highlight key risks and catalysts."
  ]
}
```

Output shape (forwarded from Dexter HTTP API):

```json
{
  "ticker": "AAPL",
  "summary": "...",
  "key_points": ["...", "..."],
  "metrics": { "pe_ratio": 28.5 },
  "raw_markdown": "## AAPL Research ..."
}
```

### 3.2 dexter.compare_companies Tool

Input:

```json
{
  "tickers": ["AAPL", "MSFT"],
  "focus": "growth_vs_margins"
}
```

Output:

```json
{
  "comparison_axis": "growth_vs_margins",
  "items": [
    { "ticker": "AAPL", "thesis": "..." },
    { "ticker": "MSFT", "thesis": "..." }
  ]
}
```

---

## 4. Go DexterService Wrapper

In `internal/infra/utcp/dexter_service.go` implement:

```go
type ResearchBundle struct {
    Ticker      string             `json:"ticker"`
    Summary     string             `json:"summary"`
    KeyPoints   []string           `json:"key_points"`
    Metrics     map[string]float64 `json:"metrics"`
    RawMarkdown string             `json:"raw_markdown"`
}

type ComparisonItem struct {
    Ticker string   `json:"ticker"`
    Thesis string   `json:"thesis"`
    Notes  []string `json:"notes"`
}

type ComparisonResult struct {
    ComparisonAxis string            `json:"comparison_axis"`
    Items          []ComparisonItem  `json:"items"`
}

type DexterService struct {
    client Client
}

func NewDexterService(c Client) *DexterService { ... }

func (s *DexterService) ResearchCompany(ctx context.Context, ticker string, questions []string) (*ResearchBundle, error) {
    input := map[string]any{
        "ticker":    ticker,
        "questions": questions,
    }
    var out ResearchBundle
    if err := s.client.CallTool(ctx, "dexter", "dexter.research_company", input, &out); err != nil {
        return nil, err
    }
    return &out, nil
}

func (s *DexterService) CompareCompanies(ctx context.Context, tickers []string, focus string) (*ComparisonResult, error) {
    input := map[string]any{
        "tickers": tickers,
        "focus":   focus,
    }
    var out ComparisonResult
    if err := s.client.CallTool(ctx, "dexter", "dexter.compare_companies", input, &out); err != nil {
        return nil, err
    }
    return &out, nil
}
```

---

## 5. Use in Jax Core

### 5.1 Attaching Research to a Trade

In `internal/app/orchestration.go`, after generating a `TradeSetup` and
`RiskResult`, the orchestrator can optionally call Dexter:

```go
if o.dexter != nil {
    bundle, err := o.dexter.ResearchCompany(ctx, setup.Symbol, []string{
        "Summarise the last 4 quarters of earnings.",
        "Highlight key risks and catalysts.",
    })
    if err == nil {
        setup.AttachResearch(bundle)
    }
}
```

Extend `TradeSetup` with a field such as:

```go
Research *ResearchBundle `json:"research,omitempty"`
```

### 5.2 Surfacing in the HTTP API

When returning a trade from the Jax HTTP API, include `research` so the UI
can display the analysis next to the numbers.

---

## 6. Use in Agent0 Lab

The lab can optionally call Dexter to annotate strategies:

- When writing reports, it can embed Dexter commentary on a few sample trades.
- When comparing strategies across sectors, it can call
  `dexter.compare_companies` to add fundamental colour to backtest results.

This is **non‑essential** to the core lab logic and can be added after basic
backtesting is working.

---

## 7. Tasks for Codex / AI

1. Spin up Dexter locally using its README instructions.  
2. Implement or adapt a small HTTP layer around Dexter providing the endpoints
   described above, if not already present.
3. Add the `dexter` provider configuration to `config/providers.json`.
4. Implement `DexterService` in Go using the UTCP client.
5. Write a small CLI or test in Go to:
   - Prompt for a ticker.
   - Call `ResearchCompany`.
   - Print a formatted summary to stdout.
6. Optionally, add a flag in Jax Core config to enable/disable Dexter usage
   (so the system can run even if Dexter is offline).
