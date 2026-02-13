# Phase 0 Validation Script
# Verifies all Phase 0 tasks are complete and working

$ErrorActionPreference = "Continue"
$script:FailureCount = 0
$script:SuccessCount = 0

function Write-Success {
    param([string]$Message)
    Write-Host "[PASS] $Message" -ForegroundColor Green
    $script:SuccessCount++
}

function Write-Failure {
    param([string]$Message)
    Write-Host "[FAIL] $Message" -ForegroundColor Red
    $script:FailureCount++
}

function Write-Info {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor Cyan
}

function Test-FileExists {
    param(
        [string]$Path,
        [string]$Description
    )
    
    if (Test-Path $Path) {
        Write-Success "$Description exists: $Path"
        return $true
    } else {
        Write-Failure "$Description missing: $Path"
        return $false
    }
}

Write-Host "================================================" -ForegroundColor Yellow
Write-Host "Phase 0 Foundation and Safety Net Validation" -ForegroundColor Yellow
Write-Host "================================================" -ForegroundColor Yellow
Write-Host ""

# Task 0.1: Golden Test Infrastructure (already complete)
Write-Info "Task 0.1: Golden Test Infrastructure"
Test-FileExists "tests\golden\cmd\capture.go" "Capture tool"
Test-FileExists "tests\golden\compare.go" "Compare tool"
Test-FileExists "tests\golden\golden_test.go" "Golden tests"
Test-FileExists "tests\golden\README.md" "Golden README"
Test-FileExists "scripts\capture-golden-baseline.ps1" "Capture script"
Test-FileExists "scripts\compare-golden-outputs.ps1" "Compare script"
Write-Host ""

# Task 0.2: Replay Harness
Write-Info "Task 0.2: Replay Harness"
Test-FileExists "tests\replay\README.md" "Replay README"
Test-FileExists "tests\replay\harness.go" "Replay harness"
Test-FileExists "tests\replay\replay_test.go" "Replay tests"
Test-FileExists "tests\replay\fixtures\aapl-rally.json" "AAPL fixture"
Test-FileExists "tests\replay\fixtures\msft-consolidation.json" "MSFT fixture"
Test-FileExists "tests\replay\fixtures\tsla-volatility.json" "TSLA fixture"
Write-Host ""

# Task 0.3: Deterministic Clock
Write-Info "Task 0.3: Deterministic Clock"
Test-FileExists "libs\testing\clock.go" "Clock implementation"
Test-FileExists "libs\testing\clock_test.go" "Clock tests"
Write-Host ""

# Task 0.4: CI Golden Test Runner
Write-Info "Task 0.4: CI Golden Test Runner"
Test-FileExists ".github\workflows\golden-tests.yml" "GitHub Actions workflow"
Test-FileExists "scripts\compare-golden-outputs.sh" "Bash compare script"
Write-Host ""

# Run Go Tests
Write-Host "================================================" -ForegroundColor Yellow
Write-Host "Running Tests" -ForegroundColor Yellow
Write-Host "================================================" -ForegroundColor Yellow
Write-Host ""

# Test 1: Golden Tests
Write-Info "Running golden tests..."
try {
    $goldenResult = go test -v ./tests/golden/ 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Success "Golden tests passed"
        Write-Host $goldenResult -ForegroundColor Gray
    } else {
        Write-Failure "Golden tests failed"
        Write-Host $goldenResult -ForegroundColor Red
    }
} catch {
    Write-Failure "Failed to run golden tests: $_"
}
Write-Host ""

# Test 2: Replay Tests
Write-Info "Running replay tests..."
try {
    $replayResult = go test -v ./tests/replay/ 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Success "Replay tests passed"
        Write-Host $replayResult -ForegroundColor Gray
    } else {
        Write-Failure "Replay tests failed"
        Write-Host $replayResult -ForegroundColor Red
    }
} catch {
    Write-Failure "Failed to run replay tests: $_"
}
Write-Host ""

# Test 3: Clock Tests
Write-Info "Running clock tests..."
try {
    $clockResult = go test -v ./libs/testing/ 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Success "Clock tests passed"
        Write-Host $clockResult -ForegroundColor Gray
    } else {
        Write-Failure "Clock tests failed"
        Write-Host $clockResult -ForegroundColor Red
    }
} catch {
    Write-Failure "Failed to run clock tests: $_"
}
Write-Host ""

# Test 4: Updated Strategy Tests (using clock)
Write-Info "Running strategy tests (with deterministic clock)..."
try {
    $strategyResult = go test -v ./libs/strategies/ 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Success "Strategy tests passed"
        Write-Host $strategyResult -ForegroundColor Gray
    } else {
        Write-Failure "Strategy tests failed"
        Write-Host $strategyResult -ForegroundColor Red
    }
} catch {
    Write-Failure "Failed to run strategy tests: $_"
}
Write-Host ""

# Verify Go Module Structure
Write-Info "Verifying Go module structure..."
try {
    go mod tidy
    go mod verify
    Write-Success "Go modules are valid"
} catch {
    Write-Failure "Go module verification failed: $_"
}
Write-Host ""

# Summary
Write-Host "================================================" -ForegroundColor Yellow
Write-Host "Validation Summary" -ForegroundColor Yellow
Write-Host "================================================" -ForegroundColor Yellow
Write-Host ""
Write-Host "Passed: $script:SuccessCount" -ForegroundColor Green
Write-Host "Failed: $script:FailureCount" -ForegroundColor Red
Write-Host ""

# Exit Criteria Check
$exitCriteriaMet = $true
$requiredFiles = @(
    "tests\golden\cmd\capture.go",
    "tests\golden\compare.go",
    "tests\replay\harness.go",
    "tests\replay\replay_test.go",
    "libs\testing\clock.go",
    "libs\testing\clock_test.go",
    ".github\workflows\golden-tests.yml"
)

foreach ($file in $requiredFiles) {
    if (-not (Test-Path $file)) {
        $exitCriteriaMet = $false
        break
    }
}

Write-Host "================================================" -ForegroundColor Yellow
Write-Host "Exit Criteria" -ForegroundColor Yellow
Write-Host "================================================" -ForegroundColor Yellow
Write-Host ""

if ($exitCriteriaMet -and $script:FailureCount -eq 0) {
    Write-Host "SUCCESS: Phase 0 Foundation COMPLETE!" -ForegroundColor Green
    Write-Host ""
    Write-Host "All tasks completed successfully:" -ForegroundColor Green
    Write-Host "  [PASS] Task 0.1: Golden Test Infrastructure" -ForegroundColor Green
    Write-Host "  [PASS] Task 0.2: Replay Harness" -ForegroundColor Green
    Write-Host "  [PASS] Task 0.3: Deterministic Clock" -ForegroundColor Green
    Write-Host "  [PASS] Task 0.4: CI Golden Test Runner" -ForegroundColor Green
    Write-Host "  [PASS] Task 0.5: Phase 0 Validation" -ForegroundColor Green
    Write-Host ""
    Write-Host "Ready to proceed to Phase 1!" -ForegroundColor Green
    exit 0
} else {
    Write-Host "WARNING: Phase 0 NOT COMPLETE" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "Please address the failures above before proceeding to Phase 1." -ForegroundColor Yellow
    exit 1
}
