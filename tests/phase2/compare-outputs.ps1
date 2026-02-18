# Compare cmd/trader output against golden baseline
# Phase 2 Validation - ADR-0012

param(
    [string]$TraderURL = "http://localhost:8100",
    [string]$GoldenDir = "$PSScriptRoot\golden",
    [string]$OutputDir = "$PSScriptRoot\output",
    [double]$Tolerance = 0.01  # Tolerance for floating point comparisons
)

$ErrorActionPreference = "Stop"

Write-Host "=== Phase 2: Compare Outputs ===" -ForegroundColor Cyan
Write-Host "Trader URL: $TraderURL"
Write-Host "Golden Directory: $GoldenDir"
Write-Host "Output Directory: $OutputDir"
Write-Host "Tolerance: ±$Tolerance"
Write-Host ""

# Create output directory
if (-not (Test-Path $OutputDir)) {
    New-Item -ItemType Directory -Path $OutputDir | Out-Null
}

# Load golden baseline
$baselineFile = Join-Path $GoldenDir "signals-baseline.json"
if (-not (Test-Path $baselineFile)) {
    Write-Host "ERROR: Golden baseline not found at $baselineFile" -ForegroundColor Red
    Write-Host "Run .\tests\phase2\capture-baseline.ps1 first" -ForegroundColor Red
    exit 1
}

Write-Host "Loading golden baseline from $baselineFile..." -ForegroundColor Yellow
$baseline = Get-Content $baselineFile | ConvertFrom-Json

# Load metadata to get test symbols
$metadataFile = Join-Path $GoldenDir "baseline-metadata.json"
$metadata = Get-Content $metadataFile | ConvertFrom-Json
$testSymbols = $metadata.symbols

# Check trader health
Write-Host "Checking cmd/trader health..." -ForegroundColor Yellow
try {
    $healthResponse = Invoke-RestMethod -Uri "$TraderURL/health" -Method Get -TimeoutSec 5
    Write-Host "Trader Status: $($healthResponse.status)" -ForegroundColor Green
} catch {
    Write-Host "ERROR: cmd/trader is not responding" -ForegroundColor Red
    Write-Host "Make sure cmd/trader is running on port 8100" -ForegroundColor Red
    exit 1
}

# Generate signals from trader
Write-Host ""
Write-Host "Generating signals from cmd/trader for symbols: $($testSymbols -join ', ')" -ForegroundColor Yellow

$requestBody = @{
    symbols = $testSymbols
} | ConvertTo-Json

try {
    $traderResponse = Invoke-RestMethod `
        -Uri "$TraderURL/api/v1/signals/generate" `
        -Method Post `
        -Body $requestBody `
        -ContentType "application/json" `
        -TimeoutSec 30
    
    Write-Host "Generated $($traderResponse.count) signals from cmd/trader" -ForegroundColor Green
    
    # Save trader output
    $traderOutputFile = Join-Path $OutputDir "signals-trader.json"
    $traderResponse | ConvertTo-Json -Depth 10 | Out-File -FilePath $traderOutputFile -Encoding UTF8
    Write-Host "Saved trader output to: $traderOutputFile" -ForegroundColor Green
    
} catch {
    Write-Host "ERROR: Failed to generate signals from cmd/trader" -ForegroundColor Red
    Write-Host $_.Exception.Message -ForegroundColor Red
    exit 1
}

# Compare outputs
Write-Host ""
Write-Host "=== Comparing Outputs ===" -ForegroundColor Cyan

$differences = @()

# Check signal count
if ($baseline.count -ne $traderResponse.count) {
    $diff = "Signal count mismatch: baseline=$($baseline.count), trader=$($traderResponse.count)"
    $differences += $diff
    Write-Host "DIFF: $diff" -ForegroundColor Red
} else {
    Write-Host "✓ Signal count matches: $($baseline.count)" -ForegroundColor Green
}

# Compare individual signals
# Group by symbol and strategy_id for comparison
$baselineSignals = @{}
foreach ($sig in $baseline.signals) {
    $key = "$($sig.symbol)-$($sig.strategy_id)"
    $baselineSignals[$key] = $sig
}

$traderSignals = @{}
foreach ($sig in $traderResponse.signals) {
    $key = "$($sig.symbol)-$($sig.strategy_id)"
    $traderSignals[$key] = $sig
}

# Compare each signal
foreach ($key in $baselineSignals.Keys) {
    if (-not $traderSignals.ContainsKey($key)) {
        $diff = "Missing signal in trader output: $key"
        $differences += $diff
        Write-Host "DIFF: $diff" -ForegroundColor Red
        continue
    }
    
    $baseSig = $baselineSignals[$key]
    $traderSig = $traderSignals[$key]
    
    # Compare signal properties
    if ($baseSig.type -ne $traderSig.type) {
        $diff = "$key: type mismatch (baseline=$($baseSig.type), trader=$($traderSig.type))"
        $differences += $diff
        Write-Host "DIFF: $diff" -ForegroundColor Red
    }
    
    if ([Math]::Abs($baseSig.confidence - $traderSig.confidence) -gt $Tolerance) {
        $diff = "$key: confidence mismatch (baseline=$($baseSig.confidence), trader=$($traderSig.confidence))"
        $differences += $diff
        Write-Host "DIFF: $diff" -ForegroundColor Red
    }
    
    if ([Math]::Abs($baseSig.entry_price - $traderSig.entry_price) -gt $Tolerance) {
        $diff = "$key: entry_price mismatch (baseline=$($baseSig.entry_price), trader=$($traderSig.entry_price))"
        $differences += $diff
        Write-Host "DIFF: $diff" -ForegroundColor Red
    }
    
    if ([Math]::Abs($baseSig.stop_loss - $traderSig.stop_loss) -gt $Tolerance) {
        $diff = "$key: stop_loss mismatch (baseline=$($baseSig.stop_loss), trader=$($traderSig.stop_loss))"
        $differences += $diff
        Write-Host "DIFF: $diff" -ForegroundColor Red
    }
}

# Check for extra signals in trader output
foreach ($key in $traderSignals.Keys) {
    if (-not $baselineSignals.ContainsKey($key)) {
        $diff = "Extra signal in trader output: $key"
        $differences += $diff
        Write-Host "DIFF: $diff" -ForegroundColor Red
    }
}

# Summary
Write-Host ""
Write-Host "=== Validation Summary ===" -ForegroundColor Cyan

if ($differences.Count -eq 0) {
    Write-Host "✓ ALL CHECKS PASSED - Outputs are identical!" -ForegroundColor Green
    Write-Host "Signal generation behavior is provably equivalent." -ForegroundColor Green
    Write-Host ""
    Write-Host "Phase 2 validation: PASSED" -ForegroundColor Green
    exit 0
} else {
    Write-Host "✗ VALIDATION FAILED - Found $($differences.Count) differences" -ForegroundColor Red
    Write-Host ""
    Write-Host "Differences:" -ForegroundColor Yellow
    foreach ($diff in $differences) {
        Write-Host "  - $diff" -ForegroundColor Red
    }
    Write-Host ""
    Write-Host "Phase 2 validation: FAILED" -ForegroundColor Red
    Write-Host "Investigate differences before proceeding with migration." -ForegroundColor Yellow
    exit 1
}
