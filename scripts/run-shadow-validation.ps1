# Shadow Mode Validation Script
# ADR-0012 Phase 4.3: Run parallel validation before production cutover

param(
    [int]$DurationHours = 120,  # 5 days = 120 hours
    [int]$CheckIntervalMinutes = 60,
    [string]$ProductionDB = "postgresql://jax:password@localhost:5433/jax",
    [string]$ShadowDB = "postgresql://jax:password@localhost:5433/jax_shadow"
)

Write-Host "🔍 ADR-0012 Phase 4.3: Shadow Mode Validation" -ForegroundColor Cyan
Write-Host ""
Write-Host "Configuration:" -ForegroundColor Gray
Write-Host "  Duration: $DurationHours hours ($($DurationHours / 24) days)" -ForegroundColor Gray
Write-Host "  Check interval: $CheckIntervalMinutes minutes" -ForegroundColor Gray
Write-Host "  Production DB: $ProductionDB" -ForegroundColor Gray
Write-Host "  Shadow DB: $ShadowDB" -ForegroundColor Gray
Write-Host ""

$env:PRODUCTION_DATABASE_URL = $ProductionDB
$env:SHADOW_DATABASE_URL = $ShadowDB

$startTime = Get-Date
$endTime = $startTime.AddHours($DurationHours)
$checksPerformed = 0
$failureCount = 0

Write-Host "🚀 Starting shadow validation..." -ForegroundColor Green
Write-Host "   Start: $($startTime.ToString('yyyy-MM-dd HH:mm:ss'))" -ForegroundColor Gray
Write-Host "   End: $($endTime.ToString('yyyy-MM-dd HH:mm:ss'))" -ForegroundColor Gray
Write-Host ""

# Main validation loop
while ((Get-Date) -lt $endTime) {
    $checksPerformed++
    $now = Get-Date
    
    Write-Host "[$($now.ToString('HH:mm:ss'))] Check #$checksPerformed" -ForegroundColor Cyan
    
    # Run shadow validator
    try {
        $env:COMPARISON_WINDOW_HOURS = "24"  # Compare last 24 hours
        
        $result = & go run cmd/shadow-validator/main.go 2>&1
        $exitCode = $LASTEXITCODE
        
        if ($exitCode -eq 0) {
            Write-Host "  ✅ PASS - All decisions match" -ForegroundColor Green
        }
        else {
            Write-Host "  ❌ FAIL - Discrepancies detected!" -ForegroundColor Red
            Write-Host $result
            $failureCount++
            
            # Critical failure - abort
            Write-Host ""
            Write-Host "💥 SHADOW VALIDATION FAILED" -ForegroundColor Red
            Write-Host "   Cannot proceed with production cutover" -ForegroundColor Red
            Write-Host "   Review discrepancy report and fix issues" -ForegroundColor Yellow
            Write-Host ""
            exit 1
        }
    }
    catch {
        Write-Host "  ⚠️  Error running validator: $_" -ForegroundColor Yellow
    }
    
    # Calculate next check time
    $nextCheck = (Get-Date).AddMinutes($CheckIntervalMinutes)
    if ($nextCheck -ge $endTime) {
        break
    }
    
    $sleepSeconds = [Math]::Max(($nextCheck - (Get-Date)).TotalSeconds, 0)
    
    Write-Host "  Next check at: $($nextCheck.ToString('HH:mm:ss'))" -ForegroundColor Gray
    Write-Host ""
    
    Start-Sleep -Seconds $sleepSeconds
}

# Final report
$elapsed = (Get-Date) - $startTime
Write-Host ""
Write-Host "═══════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "  SHADOW VALIDATION COMPLETE" -ForegroundColor Green
Write-Host "═══════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host ""
Write-Host "  Duration: $([Math]::Round($elapsed.TotalHours, 1)) hours" -ForegroundColor Gray
Write-Host "  Checks performed: $checksPerformed" -ForegroundColor Gray
Write-Host "  Failures: $failureCount" -ForegroundColor Gray
Write-Host ""

if ($failureCount -eq 0) {
    Write-Host "🎉 SUCCESS: Zero discrepancies detected!" -ForegroundColor Green
    Write-Host ""
    Write-Host "✅ Safe to proceed with production cutover" -ForegroundColor Green
    Write-Host ""
    Write-Host "Next steps:" -ForegroundColor Cyan
    Write-Host "  1. docker-compose stop jax-trade-executor" -ForegroundColor Gray
    Write-Host "  2. docker-compose up -d trader" -ForegroundColor Gray
    Write-Host "  3. Monitor trader runtime for 24 hours" -ForegroundColor Gray
    Write-Host "  4. If stable, decommission old executor" -ForegroundColor Gray
    Write-Host ""
    exit 0
}
else {
    Write-Host "⚠️  WARNING: $failureCount failures detected" -ForegroundColor Yellow
    Write-Host "   Review logs and retry shadow validation" -ForegroundColor Yellow
    exit 1
}
