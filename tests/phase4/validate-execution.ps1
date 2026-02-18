# Phase 4 - Trade Execution Module Validation Script
# Validates that the execution module is integrated and functional

$ErrorActionPreference = "Stop"

Write-Host "=== Phase 4 Execution Module Validation ===" -ForegroundColor Cyan
Write-Host ""

# Check if required services are running
function Test-ServiceRunning {
    param(
        [string]$ServiceName,
        [string]$PortOrPath
    )
    
    Write-Host "Checking $ServiceName..." -NoNewline
    
    if ($PortOrPath -match '^\d+$') {
        # Port number - check if something is listening
        $listening = Get-NetTCPConnection -LocalPort $PortOrPath -ErrorAction SilentlyContinue
        if ($listening) {
            Write-Host " OK" -ForegroundColor Green
            return $true
        }
    } else {
        # Path - check if process exists
        $process = Get-Process -Name $PortOrPath -ErrorAction SilentlyContinue
        if ($process) {
            Write-Host " OK" -ForegroundColor Green
            return $true
        }
    }
    
    Write-Host " NOT RUNNING" -ForegroundColor Yellow
    return $false
}

# Test 1: Verify execution module files exist
Write-Host "Test 1: Verify execution module files" -ForegroundColor Yellow
$files = @(
    "internal\modules\execution\engine.go",
    "internal\modules\execution\service.go",
    "internal\modules\execution\ib_adapter.go",
    "internal\modules\execution\engine_test.go"
)

$allFilesExist = $true
foreach ($file in $files) {
    $path = Join-Path "c:\Projects\jax-trading assistant" $file
    if (Test-Path $path) {
        Write-Host "  [OK] $file" -ForegroundColor Green
    } else {
        Write-Host "  [FAIL] $file MISSING" -ForegroundColor Red
        $allFilesExist = $false
    }
}

if (-not $allFilesExist) {
    Write-Host "FAILED: Not all execution module files exist" -ForegroundColor Red
    exit 1
}
Write-Host ""

# Test 2: Build cmd/trader
Write-Host "Test 2: Build cmd/trader with execution" -ForegroundColor Yellow
Push-Location "c:\Projects\jax-trading assistant"
try {
    $output = go build -o cmd/trader/trader-test.exe ./cmd/trader 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "  [OK] Build successful" -ForegroundColor Green
        $size = (Get-Item cmd/trader/trader-test.exe).Length / 1MB
        Write-Host "  Binary size: $([math]::Round($size, 2)) MB" -ForegroundColor Cyan
    } else {
        Write-Host "  [FAIL] Build failed" -ForegroundColor Red
        Write-Host $output
        exit 1
    }
} finally {
    Pop-Location
}
Write-Host ""

# Test 3: Run execution module unit tests
Write-Host "Test 3: Run execution module unit tests" -ForegroundColor Yellow
Push-Location "c:\Projects\jax-trading assistant\internal\modules\execution"
try {
    $testOutput = go test -v . 2>&1
    if ($LASTEXITCODE -eq 0) {
        # Count passed tests
        $passedTests = ($testOutput | Select-String "PASS:").Count
        Write-Host "  [OK] All tests passed ($passedTests tests)" -ForegroundColor Green
    } else {
        Write-Host "  [FAIL] Tests failed" -ForegroundColor Red
        Write-Host $testOutput
        exit 1
    }
} finally {
    Pop-Location
}
Write-Host ""

# Test 4: Verify cmd/trader exposes execution endpoint
Write-Host "Test 4: Verify handleExecute function exists" -ForegroundColor Yellow
$mainGo = Get-Content "c:\Projects\jax-trading assistant\cmd\trader\main.go" -Raw
if ($mainGo -match "handleExecute") {
    Write-Host "  [OK] handleExecute function found" -ForegroundColor Green
} else {
    Write-Host "  [FAIL] handleExecute function not found" -ForegroundColor Red
    exit 1
}

if ($mainGo -match "/api/v1/execute") {
    Write-Host "  [OK] /api/v1/execute endpoint registered" -ForegroundColor Green
} else {
    Write-Host "  [FAIL] /api/v1/execute endpoint not registered" -ForegroundColor Red
    exit 1
}
Write-Host ""

# Test 5: Check execution config parameters
Write-Host "Test 5: Verify execution config parameters" -ForegroundColor Yellow
$configParams = @(
    "IBBridgeURL",
    "ExecutionEnabled",
    "MaxRiskPerTrade",
    "MaxPositionValuePct",
    "DefaultOrderType"
)

$allParamsExist = $true
foreach ($param in $configParams) {
    if ($mainGo -match $param) {
        Write-Host "  [OK] $param config parameter found" -ForegroundColor Green
    } else {
        Write-Host "  [FAIL] $param config parameter not found" -ForegroundColor Red
        $allParamsExist = $false
    }
}

if (-not $allParamsExist) {
    Write-Host "FAILED: Not all config parameters exist" -ForegroundColor Red
    exit 1
}
Write-Host ""

# Test 6: Verify backward compatibility (old jax-trade-executor service still exists)
Write-Host "Test 6: Verify backward compatibility" -ForegroundColor Yellow
if (Test-Path "c:\Projects\jax-trading assistant\services\jax-trade-executor\cmd\jax-trade-executor\main.go") {
    Write-Host "  [OK] jax-trade-executor service preserved" -ForegroundColor Green
} else {
    Write-Host "  [WARN] jax-trade-executor service not found (acceptable if intentionally removed)" -ForegroundColor Yellow
}
Write-Host ""

# Summary
Write-Host "=== Validation Summary ===" -ForegroundColor Cyan
Write-Host "[OK] Execution module files created" -ForegroundColor Green
Write-Host "[OK] cmd/trader builds successfully" -ForegroundColor Green
Write-Host "[OK] Unit tests pass (4 test suites)" -ForegroundColor Green
Write-Host "[OK] Execution endpoint integrated" -ForegroundColor Green
Write-Host "[OK] Config parameters added" -ForegroundColor Green
Write-Host ""
Write-Host "Phase 4 validation completed successfully!" -ForegroundColor Green
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Yellow
Write-Host "  1. Start IB Bridge: .\start-ib-bridge.ps1" -ForegroundColor Gray
Write-Host "  2. Set EXECUTION_ENABLED=true in .env" -ForegroundColor Gray
Write-Host "  3. Start trader: .\start.ps1" -ForegroundColor Gray
Write-Host "  4. Test execution: POST http://localhost:8100/api/v1/execute" -ForegroundColor Gray
Write-Host ""

exit 0
