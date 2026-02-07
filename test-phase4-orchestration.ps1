# Phase 4: Orchestration Pipeline Test
# Tests the complete signal → orchestration → recommendation flow

$ErrorActionPreference = "Stop"

Write-Host "`n============================================" -ForegroundColor Cyan
Write-Host "Phase 4: Signal-to-Orchestration Pipeline Test" -ForegroundColor Cyan
Write-Host "============================================`n" -ForegroundColor Cyan

# Configuration
$baseUrl = "http://localhost:8081/api/v1"
$orchestratorUrl = "http://localhost:8091"
$signalGenUrl = "http://localhost:8096"

# Get JWT token (from previous test setup - use test credentials)
Write-Host "[1] Authenticating..." -ForegroundColor Yellow
$loginBody = @{
    username = "admin"
    password = "admin123"
} | ConvertTo-Json

try {
    $loginResponse = Invoke-RestMethod -Uri "$baseUrl/auth/login" -Method Post -Body $loginBody -ContentType "application/json"
    $token = $loginResponse.token
    Write-Host "✓ Authenticated successfully" -ForegroundColor Green
} catch {
    Write-Host "✗ Login failed (using test without auth)" -ForegroundColor Yellow
    $token = "test-token"
}

$headers = @{
    "Authorization" = "Bearer $token"
    "Content-Type" = "application/json"
}

# Step 1: Check signal generator health
Write-Host "`n[2] Checking Signal Generator status..." -ForegroundColor Yellow
try {
    $genHealth = Invoke-RestMethod -Uri "$signalGenUrl/health" -Method Get
    Write-Host "✓ Signal Generator: $($genHealth.status)" -ForegroundColor Green
    Write-Host "  Uptime: $($genHealth.uptime)" -ForegroundColor Gray
} catch {
    Write-Host "✗ Signal Generator not responding" -ForegroundColor Red
    exit 1
}

# Step 2: Check orchestrator health
Write-Host "`n[3] Checking Orchestrator status..." -ForegroundColor Yellow
try {
    $orchHealth = Invoke-RestMethod -Uri "$orchestratorUrl/health" -Method Get  
    Write-Host "✓ Orchestrator: $($orchHealth.status)" -ForegroundColor Green
} catch {
    Write-Host "✗ Orchestrator not responding" -ForegroundColor Red
    exit 1
}

# Step 3: Get recent pending signals
Write-Host "`n[4] Fetching pending signals..." -ForegroundColor Yellow
try {
    $signalsUrl = $baseUrl + '/signals?status=pending&limit=5'
    $signals = Invoke-RestMethod -Uri $signalsUrl -Method Get -Headers $headers
    Write-Host "✓ Found $($signals.total) pending signals" -ForegroundColor Green
    
    if ($signals.total -eq 0) {
        Write-Host "  No pending signals found - waiting for signal generator to run..." -ForegroundColor Yellow
        Write-Host "  Signal generator runs every 5 minutes" -ForegroundColor Gray
        exit 0
    }
    
    # Display signal details
    $signals.data | ForEach-Object {
        Write-Host "`n  Signal ID: $($_.id)" -ForegroundColor Cyan
        Write-Host "  Symbol: $($_.symbol)" -ForegroundColor White
        Write-Host "  Strategy: $($_.strategy_id)" -ForegroundColor White
        Write-Host "  Type: $($_.signal_type)" -ForegroundColor White
        Write-Host "  Confidence: $([math]::Round($_.confidence * 100, 2))%" -ForegroundColor $(if ($_.confidence -ge 0.75) { "Green" } else { "Yellow" })
        Write-Host "  Entry: `$$($_.entry_price)" -ForegroundColor White
        Write-Host "  Stop Loss: `$$($_.stop_loss)" -ForegroundColor White
        Write-Host "  Take Profit: `$$($_.take_profit)" -ForegroundColor White
        if ($_.orchestration_run_id) {
            Write-Host "  Orchestration Run: $($_.orchestration_run_id)" -ForegroundColor Magenta
        } else {
            Write-Host "  Orchestration: Not triggered" -ForegroundColor Gray
        }
    }
} catch {
    Write-Host "✗ Failed to fetch signals: $_" -ForegroundColor Red
    Write-Host $_.Exception.Message -ForegroundColor Red
}

# Step 4: Check for signals with orchestration runs
Write-Host "`n[5] Checking for orchestrated signals..." -ForegroundColor Yellow
$orchestratedCount = 0
$signals.data | ForEach-Object {
    if ($_.orchestration_run_id) {
        $orchestratedCount++
    }
}

Write-Host "✓ $orchestratedCount / $($signals.data.Count) signals have orchestration runs" -ForegroundColor $(if ($orchestratedCount -gt 0) { "Green" } else { "Yellow" })

# Step 5: Test manual orchestration trigger (if needed)
if ($signals.data.Count -gt 0) {
    $testSignal = $signals.data[0]
    
    if (-not $testSignal.orchestration_run_id) {
        Write-Host "`n[6] Testing manual orchestration trigger..." -ForegroundColor Yellow
        
        $orchestrateBody = @{
            signal_id = $testSignal.id
            symbol = $testSignal.symbol
            trigger_type = "manual_test"
            context = "Test orchestration for signal $($testSignal.id) - $($testSignal.signal_type) $($testSignal.symbol) at `$$($testSignal.entry_price)"
        } | ConvertTo-Json
        
        try {
            $orchResponse = Invoke-RestMethod -Uri "$orchestratorUrl/orchestrate" -Method Post -Body $orchestrateBody -ContentType "application/json"
            Write-Host "✓ Orchestration triggered successfully" -ForegroundColor Green
            Write-Host "  Run ID: $($orchResponse.run_id)" -ForegroundColor Cyan
            Write-Host "  Status: $($orchResponse.status)" -ForegroundColor White
            
            # Wait a bit for orchestration to process
            Write-Host "`n  Waiting 5 seconds for orchestration to process..." -ForegroundColor Gray
            Start-Sleep -Seconds 5
            
            # Check orchestration run status from database (if we had an endpoint for it)
            Write-Host "  Orchestration is running asynchronously" -ForegroundColor Gray
        } catch {
            Write-Host "✗ Failed to trigger orchestration: $_" -ForegroundColor Red
            Write-Host $_.Exception.Message -ForegroundColor Red
        }
    } else {
        Write-Host "`n[6] Signal already has orchestration run: $($testSignal.orchestration_run_id)" -ForegroundColor Cyan
    }
}

# Summary
Write-Host "`n============================================" -ForegroundColor Cyan
Write-Host "Phase 4 Test Summary" -ForegroundColor Cyan
Write-Host "============================================" -ForegroundColor Cyan
Write-Host "✓ Signal Generator: Healthy" -ForegroundColor Green
Write-Host "✓ Orchestrator: Healthy & Running" -ForegroundColor Green
Write-Host "✓ Total Pending Signals: $($signals.total)" -ForegroundColor White
Write-Host "✓ Orchestrated Signals: $orchestratedCount" -ForegroundColor White

Write-Host "`nNext Steps:" -ForegroundColor Yellow
Write-Host "- Signals with confidence ≥ 75% should auto-trigger orchestration" -ForegroundColor Gray  
Write-Host "- Wait for signal generator's next run (every 5 minutes)" -ForegroundColor Gray
Write-Host "- Check orchestration_runs table in database for AI analysis results" -ForegroundColor Gray
Write-Host "- Use signal approval API to approve/reject signals" -ForegroundColor Gray
Write-Host "`n"
