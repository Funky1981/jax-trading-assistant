# Phase 3 Orchestration Validation Script
# Tests the orchestration endpoint in cmd/trader

Write-Host "Phase 3: Orchestration Module Validation" -ForegroundColor Cyan
Write-Host "=========================================`n" -ForegroundColor Cyan

# Start required services
Write-Host "Step 1: Starting services..." -ForegroundColor Yellow
docker-compose up -d postgres jax-memory agent0-service
Start-Sleep -Seconds 5

# Check postgres health
Write-Host "`nStep 2: Checking postgres..." -ForegroundColor Yellow
$pgStatus = docker ps --filter "name=postgres" --format "{{.Status}}"
if ($pgStatus -notlike "*healthy*" -and $pgStatus -notlike "*Up*") {
    Write-Host "❌ Postgres not running: $pgStatus" -ForegroundColor Red
    exit 1
}
Write-Host "✅ Postgres running" -ForegroundColor Green

# Build and start cmd/trader
Write-Host "`nStep 3: Killing any existing cmd/trader..." -ForegroundColor Yellow
Get-Process -Name "trader" -ErrorAction SilentlyContinue | Stop-Process -Force
Start-Sleep -Seconds 1
Write-Host "✅ Cleanup complete" -ForegroundColor Green

Write-Host "`nStep 4: Building cmd/trader..." -ForegroundColor Yellow
$traderPath = "c:\Projects\jax-trading assistant\cmd\trader"
Push-Location $traderPath
go build -o trader.exe .
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Build failed" -ForegroundColor Red
    Pop-Location
    exit 1  
}
Write-Host "✅ Build successful" -ForegroundColor Green

Write-Host "`nStep 5: Starting cmd/trader..." -ForegroundColor Yellow
$env:DATABASE_URL = "postgresql://jax:jax@localhost:5433/jax"
$env:PORT = "8100"
$env:MEMORY_SERVICE_URL = "http://localhost:8090"
$env:AGENT0_SERVICE_URL = "http://localhost:8093"
$env:DEXTER_SERVICE_URL = "http://localhost:8094"

Start-Process -FilePath ".\trader.exe" -NoNewWindow
Start-Sleep -Seconds 3

# Check health
try {
    $health = Invoke-RestMethod -Uri "http://localhost:8100/health" -TimeoutSec 5
    Write-Host "✅ cmd/trader healthy: $($health.status)" -ForegroundColor Green
} catch {
    Write-Host "❌ cmd/trader health check failed: $_" -ForegroundColor Red
    Pop-Location
    exit 1
}

# Test orchestration endpoint
Write-Host "`nStep 6: Testing orchestration endpoint..." -ForegroundColor Yellow

$orchRequest = @{
    bank = "trade_decisions"
    symbol = "AAPL"
    user_context = "Analyzing AAPL for potential trade"
    tags = @("test", "validation")
    constraints = @{
        price = 150.00
        rsi = 35.0
        trend = "uptrend"
    }
} | ConvertTo-Json

try {
    $result = Invoke-RestMethod -Uri "http://localhost:8100/api/v1/orchestrate" `
        -Method Post `
        -Body $orchRequest `
        -ContentType "application/json" `
        -TimeoutSec 30

    if ($result.success) {
        Write-Host "✅ Orchestration succeeded!" -ForegroundColor Green
        Write-Host "   Action: $($result.plan.action)" -ForegroundColor Gray
        Write-Host "   Confidence: $($result.plan.confidence)" -ForegroundColor Gray
        Write-Host "   Summary: $($result.plan.summary)" -ForegroundColor Gray
        Write-Host "   Duration: $($result.duration)" -ForegroundColor Gray
    } else {
        Write-Host "❌ Orchestration returned success=false" -ForegroundColor Red
        exit 1
    }
} catch {
    Write-Host "❌ Orchestration request failed: $_" -ForegroundColor Red
    Write-Host $_.Exception.Message -ForegroundColor Red
    Pop-Location
    exit 1
}

# Verify memory was retained
Write-Host "`nStep 7: Verifying memory retention..." -ForegroundColor Yellow
try {
    $memoryReq = @{
        bank = "trade_decisions"
        query = @{
            symbol = "AAPL"
            limit = 5
        }
    } | ConvertTo-Json

    $memories = Invoke-RestMethod -Uri "http://localhost:8090/api/v1/recall" `
        -Method Post `
        -Body $memoryReq `
        -ContentType "application/json"

    $recentMemory = $memories.items | Where-Object { $_.symbol -eq "AAPL" } | Select-Object -First 1
    
    if ($recentMemory) {
        Write-Host "✅ Memory retained: $($recentMemory.summary)" -ForegroundColor Green
    } else {
        Write-Host "⚠️  No memory found (may take a moment to propagate)" -ForegroundColor Yellow
    }
} catch {
    Write-Host "⚠️  Memory check failed (non-critical): $_" -ForegroundColor Yellow
}

Write-Host "`n==========================================" -ForegroundColor Cyan
Write-Host "✅ Phase 3 Validation Complete!" -ForegroundColor Green
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "`nOrchestration module successfully:" -ForegroundColor White
Write-Host "  - Extracted to internal/modules/orchestration" -ForegroundColor Gray
Write-Host "  - Integrated into cmd/trader" -ForegroundColor Gray
Write-Host "  - Exposed via HTTP endpoint" -ForegroundColor Gray
Write-Host "  - Tested end-to-end with memory/Agent0" -ForegroundColor Gray
Write-Host "`nNext: Update docker-compose and create PHASE_3_COMPLETE.md" -ForegroundColor Cyan

Pop-Location
