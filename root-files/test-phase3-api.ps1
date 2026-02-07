# Phase 3 API Testing Script
Write-Host "`n===== Phase 3: Signal Approval API Testing =====" -ForegroundColor Cyan

# BaseURL
$baseUrl = "http://localhost:8081"

# Step 1: Login
Write-Host "`n[1/5] Testing LOGIN..." -ForegroundColor Yellow
try {
    $loginBody = @{
        username = "admin"
        password = "admin"
    } | ConvertTo-Json

    $loginResponse = Invoke-RestMethod -Uri "$baseUrl/auth/login" -Method POST -Body $loginBody -ContentType "application/json"
    $token = $loginResponse.token
    Write-Host "✓ Login successful, token obtained" -ForegroundColor Green
} catch {
    Write-Host "✗ Login failed: $_" -ForegroundColor Red
    exit 1
}

# Set auth headers
$headers = @{
    "Authorization" = "Bearer $token"
    "Content-Type" = "application/json"
}

# Step 2: List signals
Write-Host "`n[2/5] Testing GET /api/v1/signals..." -ForegroundColor Yellow
try {
    $uri = "$baseUrl/api/v1/signals" + '?status=pending&limit=5'
    $listResponse = Invoke-RestMethod -Uri $uri -Headers $headers -Method GET
    Write-Host "✓ GET /api/v1/signals - Status: 200" -ForegroundColor Green
    Write-Host "  Total signals: $($listResponse.total)" -ForegroundColor Cyan
    Write-Host "  Returned: $($listResponse.signals.Count) signals" -ForegroundColor Cyan
    
    if ($listResponse.signals.Count -gt 0) {
        $testSignal = $listResponse.signals[0]
        $signalId = $testSignal.id
        Write-Host "  Sample signal: $($testSignal.symbol) $($testSignal.signal_type) @ $($testSignal.entry_price)" -ForegroundColor Cyan
    }
} catch {
    Write-Host "✗ GET /api/v1/signals failed: $_" -ForegroundColor Red
    exit 1
}

# Step 3: Get single signal
Write-Host "`n[3/5] Testing GET /api/v1/signals/{id}..." -ForegroundColor Yellow
try {
    $getResponse = Invoke-RestMethod -Uri "$baseUrl/api/v1/signals/$signalId" -Headers $headers -Method GET
    Write-Host "✓ GET /api/v1/signals/$signalId - Status: 200" -ForegroundColor Green
    Write-Host "  Signal: $($getResponse.symbol) $($getResponse.signal_type)" -ForegroundColor Cyan
    Write-Host "  Entry: $($getResponse.entry_price), Stop: $($getResponse.stop_loss), Target: $($getResponse.take_profit)" -ForegroundColor Cyan
    Write-Host "  Confidence: $($getResponse.confidence), Status: $($getResponse.status)" -ForegroundColor Cyan
} catch {
    Write-Host "✗ GET /api/v1/signals/{id} failed: $_" -ForegroundColor Red
    exit 1
}

# Step 4: Approve signal
Write-Host "`n[4/5] Testing POST /api/v1/signals/{id}/approve..." -ForegroundColor Yellow
try {
    $approveBody = @{
        approved_by = "test-script"
        modification_notes = "Phase 3 testing - approved via API"
    } | ConvertTo-Json

    $approveResponse = Invoke-RestMethod -Uri "$baseUrl/api/v1/signals/$signalId/approve" -Headers $headers -Method POST -Body $approveBody
    Write-Host "✓ POST /api/v1/signals/$signalId/approve - Status: 200" -ForegroundColor Green
    Write-Host "  Signal status updated to: $($approveResponse.status)" -ForegroundColor Cyan
} catch {
    Write-Host "✗ POST /api/v1/signals/{id}/approve failed: $_" -ForegroundColor Red
    exit 1
}

# Step 5: Reject another signal
Write-Host "`n[5/5] Testing POST /api/v1/signals/{id}/reject..." -ForegroundColor Yellow
try {
    # Get another pending signal
    $uri2 = "$baseUrl/api/v1/signals" + '?status=pending&limit=1'
    $listResponse2 = Invoke-RestMethod -Uri $uri2 -Headers $headers -Method GET
    
    if ($listResponse2.signals.Count -gt 0) {
        $rejectSignalId = $listResponse2.signals[0].id
        
        $rejectBody = @{
            approved_by = "test-script"
            rejection_reason = "Phase 3 testing - rejected via API"
        } | ConvertTo-Json

        $rejectResponse = Invoke-RestMethod -Uri "$baseUrl/api/v1/signals/$rejectSignalId/reject" -Headers $headers -Method POST -Body $rejectBody
        Write-Host "✓ POST /api/v1/signals/$rejectSignalId/reject - Status: 200" -ForegroundColor Green
        Write-Host "  Signal status updated to: $($rejectResponse.status)" -ForegroundColor Cyan
    } else {
        Write-Host "⚠ No pending signals available for rejection test" -ForegroundColor Yellow
    }
} catch {
    Write-Host "✗ POST /api/v1/signals/{id}/reject failed: $_" -ForegroundColor Red
    exit 1
}

# Verify database state
Write-Host "`n===== Database Verification =====" -ForegroundColor Cyan
Write-Host "Signal counts by status:" -ForegroundColor Cyan
$dbOutput = docker compose -f root-files/docker-compose.yml exec postgres psql -U jax -d jax -t -c 'SELECT status, COUNT(*) FROM strategy_signals GROUP BY status ORDER BY status;' 2>$null
Write-Host $dbOutput -ForegroundColor White

Write-Host "`n===== Phase 3 API Testing COMPLETE =====" -ForegroundColor Green
Write-Host "All 4 endpoints tested successfully!" -ForegroundColor Green
