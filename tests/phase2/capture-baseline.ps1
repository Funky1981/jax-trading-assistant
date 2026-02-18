# Capture Golden Baseline from jax-signal-generator
# Phase 2 Validation - ADR-0012

param(
    [string]$ServiceURL = "http://localhost:8096",
    [string]$OutputDir = "$PSScriptRoot\golden"
)

$ErrorActionPreference = "Stop"

Write-Host "=== Phase 2: Capture Golden Baseline ===" -ForegroundColor Cyan
Write-Host "Service URL: $ServiceURL"
Write-Host "Output Directory: $OutputDir"
Write-Host ""

# Create output directory
if (-not (Test-Path $OutputDir)) {
    New-Item -ItemType Directory -Path $OutputDir | Out-Null
    Write-Host "Created output directory: $OutputDir" -ForegroundColor Green
}

# Test symbols for validation
$testSymbols = @("AAPL", "MSFT", "GOOGL")

# Check service health
Write-Host "Checking jax-signal-generator health..." -ForegroundColor Yellow
try {
    $healthResponse = Invoke-RestMethod -Uri "$ServiceURL/health" -Method Get -TimeoutSec 5
    Write-Host "Service Status: $($healthResponse.status)" -ForegroundColor Green
} catch {
    Write-Host "ERROR: jax-signal-generator is not responding" -ForegroundColor Red
    Write-Host "Make sure the service is running: docker-compose up -d jax-signal-generator" -ForegroundColor Red
    exit 1
}

# Generate signals
Write-Host ""
Write-Host "Generating signals for symbols: $($testSymbols -join ', ')" -ForegroundColor Yellow

$requestBody = @{
    symbols = $testSymbols
} | ConvertTo-Json

try {
    $response = Invoke-RestMethod `
        -Uri "$ServiceURL/api/v1/signals/generate" `
        -Method Post `
        -Body $requestBody `
        -ContentType "application/json" `
        -TimeoutSec 30
    
    Write-Host "Generated $($response.count) signals in $($response.duration)" -ForegroundColor Green
    
    # Save to golden baseline
    $outputFile = Join-Path $OutputDir "signals-baseline.json"
    $response | ConvertTo-Json -Depth 10 | Out-File -FilePath $outputFile -Encoding UTF8
    Write-Host "Saved baseline to: $outputFile" -ForegroundColor Green
    
    # Save metadata
    $metadata = @{
        timestamp = Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ"
        service = "jax-signal-generator"
        service_url = $ServiceURL
        symbols = $testSymbols
        signal_count = $response.count
        duration = $response.duration
    }
    
    $metadataFile = Join-Path $OutputDir "baseline-metadata.json"
    $metadata | ConvertTo-Json -Depth 5 | Out-File -FilePath $metadataFile -Encoding UTF8
    Write-Host "Saved metadata to: $metadataFile" -ForegroundColor Green
    
    # Display summary
    Write-Host ""
    Write-Host "=== Baseline Summary ===" -ForegroundColor Cyan
    Write-Host "Signals: $($response.count)"
    Write-Host "Duration: $($response.duration)"
    
    if ($response.signals -and $response.signals.Count -gt 0) {
        Write-Host ""
        Write-Host "Sample Signals:" -ForegroundColor Yellow
        foreach ($signal in $response.signals | Select-Object -First 3) {
            Write-Host "  - $($signal.symbol): $($signal.type) @ $($signal.entry_price) (confidence: $($signal.confidence))"
        }
    }
    
    Write-Host ""
    Write-Host "Golden baseline captured successfully!" -ForegroundColor Green
    exit 0
    
} catch {
    Write-Host "ERROR: Failed to generate signals" -ForegroundColor Red
    Write-Host $_.Exception.Message -ForegroundColor Red
    exit 1
}
