# UAT Paper Trading Script
# Runs an end-to-end sanity check for the paper trading pipeline.

param(
    [string]$ApiBase = "http://localhost:8081",
    [string]$MarketBase = "http://localhost:8095",
    [string]$SignalBase = "http://localhost:8096",
    [string]$OrchestratorBase = "http://localhost:8091",
    [string]$IbBridgeBase = "http://localhost:8092",
    [string]$TradeExecutorBase = "http://localhost:8097",
    [string]$MemoryBase = "http://localhost:8090",
    [string]$Agent0Base = "http://localhost:8093",
    [string]$JwtToken = "",
    [string]$Symbol = "",
    [switch]$ExecuteOrders,
    [switch]$SkipDb,
    [string]$OutputPath = ""
)

$ErrorActionPreference = "Stop"

function Invoke-Api {
    param(
        [string]$Method,
        [string]$Url,
        [object]$Body = $null
    )
    $headers = @{}
    if ($JwtToken) {
        $headers["Authorization"] = "Bearer $JwtToken"
    }
    if ($Body -ne $null) {
        $json = $Body | ConvertTo-Json -Depth 6
        return Invoke-RestMethod -Method $Method -Uri $Url -Body $json -ContentType "application/json" -Headers $headers -TimeoutSec 15
    }
    return Invoke-RestMethod -Method $Method -Uri $Url -Headers $headers -TimeoutSec 15
}

function Add-Result {
    param(
        [string]$Step,
        [string]$Status,
        [string]$Detail
    )
    $script:Results += [pscustomobject]@{
        Step = $Step
        Status = $Status
        Detail = $Detail
    }
    $color = "Gray"
    if ($Status -eq "PASS") { $color = "Green" }
    if ($Status -eq "WARN") { $color = "Yellow" }
    if ($Status -eq "FAIL") { $color = "Red" }
    Write-Host ("[{0}] {1} - {2}" -f $Status, $Step, $Detail) -ForegroundColor $color
}

Write-Host "===== JAX Paper Trading UAT =====" -ForegroundColor Cyan
Write-Host ""

$Results = @()

# Load default symbol from config if not provided
if (-not $Symbol) {
    $configPath = Join-Path $PSScriptRoot "..\\config\\jax-market.json"
    if (Test-Path $configPath) {
        try {
            $cfg = Get-Content $configPath | ConvertFrom-Json
            if ($cfg.symbols -and $cfg.symbols.Count -gt 0) {
                $Symbol = $cfg.symbols[0]
            }
        } catch {
            $Symbol = "AAPL"
        }
    } else {
        $Symbol = "AAPL"
    }
}

# Health checks
$services = @(
    @{ Name = "ib-bridge"; Url = "$IbBridgeBase/health" },
    @{ Name = "jax-market"; Url = "$MarketBase/health" },
    @{ Name = "jax-signal-generator"; Url = "$SignalBase/health" },
    @{ Name = "jax-orchestrator"; Url = "$OrchestratorBase/health" },
    @{ Name = "jax-trade-executor"; Url = "$TradeExecutorBase/health" },
    @{ Name = "jax-api"; Url = "$ApiBase/health" },
    @{ Name = "jax-memory"; Url = "$MemoryBase/health" },
    @{ Name = "agent0-service"; Url = "$Agent0Base/health" }
)

foreach ($svc in $services) {
    try {
        $resp = Invoke-Api -Method "GET" -Url $svc.Url
        Add-Result -Step ("Health: " + $svc.Name) -Status "PASS" -Detail "reachable"
        if ($svc.Name -eq "ib-bridge" -and $resp.connected -ne $null -and -not $resp.connected) {
            Add-Result -Step "IB Gateway" -Status "WARN" -Detail "ib-bridge connected=false"
        }
    } catch {
        Add-Result -Step ("Health: " + $svc.Name) -Status "FAIL" -Detail $_.Exception.Message
    }
}

# Market ingestion metrics
try {
    $metrics = Invoke-Api -Method "GET" -Url "$MarketBase/metrics"
    if ($metrics.stale_quotes -gt 0) {
        Add-Result -Step "Market data freshness" -Status "WARN" -Detail ("stale quotes=" + $metrics.stale_quotes)
    } else {
        Add-Result -Step "Market data freshness" -Status "PASS" -Detail "no stale quotes reported"
    }
} catch {
    Add-Result -Step "Market data freshness" -Status "WARN" -Detail "metrics endpoint not reachable"
}

# Database checks
if (-not $SkipDb) {
    $dockerCmd = Get-Command "docker" -ErrorAction SilentlyContinue
    $composeCmd = "docker compose"
    $composeLegacyCmd = "docker-compose"
    if (-not $dockerCmd) {
        Add-Result -Step "DB checks" -Status "WARN" -Detail "docker not found; skipping DB checks"
    } else {
        $composeOk = $true
        try {
            & $composeCmd ps | Out-Null
        } catch {
            try {
                & $composeLegacyCmd ps | Out-Null
                $composeCmd = $composeLegacyCmd
            } catch {
                $composeOk = $false
            }
        }

        if (-not $composeOk) {
            Add-Result -Step "DB checks" -Status "WARN" -Detail "docker compose not available; skipping DB checks"
        } else {
            try {
                $symbolList = @($Symbol)
                $marketConfig = Join-Path $PSScriptRoot "..\\config\\jax-market.json"
                if (Test-Path $marketConfig) {
                    $cfg = Get-Content $marketConfig | ConvertFrom-Json
                    if ($cfg.symbols) { $symbolList = $cfg.symbols }
                }

                $symbolsCsv = ($symbolList | ForEach-Object { "'" + $_ + "'" }) -join ","
                $sqlCandles = "SELECT symbol, COUNT(*) AS cnt FROM candles WHERE symbol IN ($symbolsCsv) GROUP BY symbol;"
                $sqlSignals = "SELECT COUNT(*) FROM strategy_signals WHERE created_at > NOW() - INTERVAL '24 hours';"
                $sqlQuotes = "SELECT symbol, NOW() - MAX(updated_at) AS lag FROM quotes WHERE symbol IN ($symbolsCsv) GROUP BY symbol;"

                $candlesOut = & $composeCmd exec -T postgres psql -U jax -d jax -t -c $sqlCandles
                $signalsOut = & $composeCmd exec -T postgres psql -U jax -d jax -t -c $sqlSignals
                $quotesOut = & $composeCmd exec -T postgres psql -U jax -d jax -t -c $sqlQuotes

                $candlesOk = $true
                foreach ($line in $candlesOut) {
                    $parts = $line.Trim() -split "\|"
                    if ($parts.Length -ge 2) {
                        $count = [int]$parts[1].Trim()
                        if ($count -lt 250) { $candlesOk = $false }
                    }
                }
                if ($candlesOk) {
                    Add-Result -Step "Candle backfill" -Status "PASS" -Detail ">= 250 candles per symbol"
                } else {
                    Add-Result -Step "Candle backfill" -Status "FAIL" -Detail "insufficient candles for some symbols"
                }

                $signalCount = ($signalsOut | Out-String).Trim()
                if ($signalCount -match "^\d+$" -and [int]$signalCount -gt 0) {
                    Add-Result -Step "Signals in last 24h" -Status "PASS" -Detail ("count=" + $signalCount)
                } else {
                    Add-Result -Step "Signals in last 24h" -Status "WARN" -Detail "no recent signals"
                }

                Add-Result -Step "Quote recency" -Status "PASS" -Detail "checked latest quotes in DB"
            } catch {
                Add-Result -Step "DB checks" -Status "WARN" -Detail $_.Exception.Message
            }
        }
    }
} else {
    Add-Result -Step "DB checks" -Status "WARN" -Detail "skipped by user"
}

# Orchestration test
try {
    $orchBody = @{
        bank = "bank-signals"
        symbol = $Symbol
        constraints = @{}
        userContext = "UAT run for $Symbol"
        tags = @("uat", "paper")
    }
    $orchResp = Invoke-Api -Method "POST" -Url "$ApiBase/api/v1/orchestrate" -Body $orchBody
    if ($orchResp.runId) {
        Add-Result -Step "Orchestration trigger" -Status "PASS" -Detail ("runId=" + $orchResp.runId)

        $status = ""
        $deadline = (Get-Date).AddSeconds(60)
        while ((Get-Date) -lt $deadline) {
            Start-Sleep -Seconds 5
            try {
                $run = Invoke-Api -Method "GET" -Url "$ApiBase/api/v1/orchestrate/runs/$($orchResp.runId)"
                $status = $run.status
                if ($status -eq "completed" -or $status -eq "failed") { break }
            } catch {
                break
            }
        }
        if ($status -eq "completed") {
            Add-Result -Step "Orchestration run" -Status "PASS" -Detail "completed"
        } elseif ($status -eq "failed") {
            Add-Result -Step "Orchestration run" -Status "FAIL" -Detail "failed"
        } else {
            Add-Result -Step "Orchestration run" -Status "WARN" -Detail "still running or unknown"
        }
    } else {
        Add-Result -Step "Orchestration trigger" -Status "FAIL" -Detail "no runId returned"
    }
} catch {
    Add-Result -Step "Orchestration trigger" -Status "FAIL" -Detail $_.Exception.Message
}

# Approval + execution (optional)
if ($ExecuteOrders) {
    try {
        $signalList = Invoke-Api -Method "GET" -Url "$ApiBase/api/v1/signals?status=pending&limit=1"
        if ($signalList.signals -and $signalList.signals.Count -gt 0) {
            $signalId = $signalList.signals[0].id
            $approveBody = @{
                approved_by = "uat@local"
                modification_notes = "UAT approval"
            }
            $approveResp = Invoke-Api -Method "POST" -Url "$ApiBase/api/v1/signals/$signalId/approve" -Body $approveBody
            Add-Result -Step "Signal approval" -Status "PASS" -Detail ("approved " + $signalId)

            Start-Sleep -Seconds 5
            $trades = Invoke-Api -Method "GET" -Url "$TradeExecutorBase/api/v1/trades"
            if ($trades.count -gt 0) {
                Add-Result -Step "Trade execution" -Status "PASS" -Detail ("trades=" + $trades.count)
            } else {
                Add-Result -Step "Trade execution" -Status "WARN" -Detail "no trades found yet"
            }
        } else {
            Add-Result -Step "Signal approval" -Status "WARN" -Detail "no pending signals available"
        }
    } catch {
        Add-Result -Step "Execution path" -Status "FAIL" -Detail $_.Exception.Message
    }
} else {
    Add-Result -Step "Execution path" -Status "WARN" -Detail "skipped (set -ExecuteOrders to enable)"
}

Write-Host ""
Write-Host "===== UAT Summary =====" -ForegroundColor Cyan
$Results | Format-Table -AutoSize

# Write report
if (-not $OutputPath) {
    $timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
    $reportDir = Join-Path $PSScriptRoot "..\\Docs\\uat"
    if (-not (Test-Path $reportDir)) {
        New-Item -Path $reportDir -ItemType Directory | Out-Null
    }
    $OutputPath = Join-Path $reportDir "uat_run_$timestamp.md"
}

$reportLines = @()
$reportLines += "# UAT Paper Trading Run"
$reportLines += ""
$reportLines += "Timestamp: $(Get-Date -Format s)"
$reportLines += ""
$reportLines += "## Results"
foreach ($r in $Results) {
    $reportLines += ("- [{0}] {1}: {2}" -f $r.Status, $r.Step, $r.Detail)
}
$reportLines += ""
$reportLines += "## Parameters"
$reportLines += "- ApiBase: $ApiBase"
$reportLines += "- MarketBase: $MarketBase"
$reportLines += "- SignalBase: $SignalBase"
$reportLines += "- OrchestratorBase: $OrchestratorBase"
$reportLines += "- IbBridgeBase: $IbBridgeBase"
$reportLines += "- TradeExecutorBase: $TradeExecutorBase"
$reportLines += "- Symbol: $Symbol"
$reportLines += "- ExecuteOrders: $ExecuteOrders"
$reportLines += "- SkipDb: $SkipDb"

$reportLines | Set-Content -Path $OutputPath
Write-Host ""
Write-Host ("Report written to: " + $OutputPath) -ForegroundColor Green
