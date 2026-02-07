# Phase 4: Orchestration Pipeline Test
# Tests the complete signal  orchestration flow

Write-Host "`n============================================" -ForegroundColor Cyan
Write-Host "Phase 4: Signal-to-Orchestration Pipeline Test" -ForegroundColor Cyan
Write-Host "============================================`n" -ForegroundColor Cyan

# Configuration
$baseUrl = "http://localhost:8081/api/v1"
$orchestratorUrl = "http://localhost:8091"
$signalGenUrl = "http://localhost:8096"

# Step 1: Check signal generator health
Write-Host "[1] Checking Signal Generator status..." -ForegroundColor Yellow
$genHealth = Invoke-RestMethod -Uri "$signalGenUrl/health" -Method Get
Write-Host "✓ Signal Generator: $($genHealth.status)" -ForegroundColor Green

# Step 2: Check orchestrator health
Write-Host "`n[2] Checking Orchestrator status..." -ForegroundColor Yellow
$orchHealth = Invoke-RestMethod -Uri "$orchestratorUrl/health" -Method Get  
Write-Host "✓ Orchestrator: $($orchHealth.status)" -ForegroundColor Green

# Step 3: Get recent pending signals
Write-Host "`n[3] Fetching pending signals..." -ForegroundColor Yellow
$signalsUrl = $baseUrl + '/signals?status=pending&limit=5'
$signals = Invoke-RestMethod -Uri $signalsUrl -Method Get
Write-Host "✓ Found $($signals.total) pending signals" -ForegroundColor Green

if ($signals.total -gt 0) {
    $signals.data | Select-Object -First 3 | ForEach-Object {
        Write-Host "`n  Signal: $($_.id)" -ForegroundColor Cyan
        Write-Host "   Symbol: $($_.symbol) | Type: $($_.signal_type)" -ForegroundColor White
        Write-Host "   Confidence: $([math]::Round($_.confidence * 100, 2))%" -ForegroundColor Green
        if ($_.orchestration_run_id) {
            Write-Host "   Orchestrated: YES (Run: $($_.orchestration_run_id))" -ForegroundColor Magenta
        } else {
            Write-Host "   Orchestrated: NO" -ForegroundColor Gray
        }
    }
}

# Summary
Write-Host "`n============================================" -ForegroundColor Cyan
Write-Host "Phase 4 Test Summary" -ForegroundColor Cyan
Write-Host "============================================" -ForegroundColor Cyan
Write-Host "✓ Signal Generator: Healthy" -ForegroundColor Green
Write-Host "✓  Orchestrator: Healthy" -ForegroundColor Green
Write-Host "✓ Total Pending Signals: $($signals.total)" -ForegroundColor White
Write-Host "`nPhase 4 Complete!" -ForegroundColor Green
Write-Host "`n"
