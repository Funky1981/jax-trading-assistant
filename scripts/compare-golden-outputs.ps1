# Compare Golden Outputs
# This script compares current system behavior against the golden baseline

param(
    [string]$BaselineDate = "2026-02-13"
)

Write-Host "🔍 Comparing Against Golden Baseline..." -ForegroundColor Cyan
Write-Host "Baseline date: $BaselineDate`n" -ForegroundColor Gray

$exitCode = 0
$categories = @("signals", "executions", "orchestration")

foreach ($category in $categories) {
    $baselineFile = "tests\golden\$category\baseline-$BaselineDate.json"
    $currentFile = "tests\golden\$category\current.json"
    
    Write-Host "Checking $category..." -ForegroundColor Yellow
    
    # Check if baseline exists
    if (-not (Test-Path $baselineFile)) {
        Write-Host "  ⚠️  Baseline file not found: $baselineFile" -ForegroundColor Yellow
        Write-Host "      Run: .\scripts\capture-golden-baseline.ps1" -ForegroundColor Gray
        continue
    }
    
    # Check if current exists
    if (-not (Test-Path $currentFile)) {
        Write-Host "  ⚠️  Current snapshot not found: $currentFile" -ForegroundColor Yellow
        Write-Host "      Capturing current state..." -ForegroundColor Gray
        
        # Capture current
        go run .\tests\golden\cmd\capture.go -output "tests\golden\$category" 2>&1 | Out-Null
        
        if (-not (Test-Path $currentFile)) {
            Write-Host "  ❌ Failed to capture current state" -ForegroundColor Red
            $exitCode = 1
            continue
        }
    }
    
    # Load and compare JSON
    try {
        $baseline = Get-Content $baselineFile -Raw | ConvertFrom-Json
        $current = Get-Content $currentFile -Raw | ConvertFrom-Json
        
        # Compare response structures (ignoring timestamps and IDs)
        $baselineResponse = $baseline.response | ConvertTo-Json -Depth 10 -Compress
        $currentResponse = $current.response | ConvertTo-Json -Depth 10 -Compress
        
        if ($baselineResponse -eq $currentResponse) {
            Write-Host "  ✅ Match" -ForegroundColor Green
        } else {
            Write-Host "  ❌ Mismatch detected" -ForegroundColor Red
            
            # Show diff summary
            $baselineCount = 0
            $currentCount = 0
            
            if ($baseline.response -is [array]) {
                $baselineCount = $baseline.response.Count
            }
            if ($current.response -is [array]) {
                $currentCount = $current.response.Count
            }
            
            Write-Host "      Baseline count: $baselineCount" -ForegroundColor Gray
            Write-Host "      Current count:  $currentCount" -ForegroundColor Gray
            
            # Save diff for review
            $diffFile = "tests\golden\$category\diff-$(Get-Date -Format 'yyyy-MM-dd-HHmmss').json"
            @{
                baseline = $baseline
                current = $current
                diff_summary = @{
                    baseline_count = $baselineCount
                    current_count = $currentCount
                }
            } | ConvertTo-Json -Depth 10 | Out-File $diffFile
            
            Write-Host "      Diff saved to: $diffFile" -ForegroundColor Gray
            $exitCode = 1
        }
    } catch {
        Write-Host "  ❌ Error comparing: $_" -ForegroundColor Red
        $exitCode = 1
    }
}

Write-Host ""
if ($exitCode -eq 0) {
    Write-Host "🎉 All golden tests passed!" -ForegroundColor Green
} else {
    Write-Host "❌ Some golden tests failed" -ForegroundColor Red
    Write-Host "`nIf changes are intentional:" -ForegroundColor Yellow
    Write-Host "  1. Review the differences carefully"
    Write-Host "  2. Update baseline: .\scripts\capture-golden-baseline.ps1"
    Write-Host "  3. Commit the new baseline with explanation"
}

exit $exitCode
