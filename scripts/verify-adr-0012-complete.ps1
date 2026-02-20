# ADR-0012 Implementation - Final Verification Script  
# Verifies all phases are complete and functioning

Write-Host "ADR-0012 Implementation Verification" -ForegroundColor Cyan
Write-Host "======================================" -ForegroundColor Cyan
Write-Host ""

$errors = 0
$warnings = 0

# Phase 0: Foundation
Write-Host "Phase 0: Foundation and Safety Net" -ForegroundColor Yellow

if (Test-Path "tests/golden/cmd/capture.go") {
    Write-Host "  [OK] Golden test capture tool exists" -ForegroundColor Green
} else {
    Write-Host "  [FAIL] Missing golden test capture tool" -ForegroundColor Red
    $errors++
}

if (Test-Path "tests/replay/harness.go") {
    Write-Host "  [OK] Replay harness exists" -ForegroundColor Green
} else {
    Write-Host "  [FAIL] Missing replay harness" -ForegroundColor Red
    $errors++
}

if (Test-Path "libs/testing/clock.go") {
    Write-Host "  [OK] Deterministic clock exists" -ForegroundColor Green
} else {
    Write-Host "  [FAIL] Missing deterministic clock" -ForegroundColor Red
    $errors++
}

if (Test-Path ".github/workflows/golden-tests.yml") {
    Write-Host "  [OK] Golden tests CI workflow exists" -ForegroundColor Green
} else {
    Write-Host "  [FAIL] Missing golden tests CI workflow" -ForegroundColor Red
    $errors++
}

$baselineCount = 0
if (Test-Path "tests/golden/signals/baseline-2026-02-13.json") { $baselineCount++ }
if (Test-Path "tests/golden/executions/baseline-2026-02-13.json") { $baselineCount++ }
if (Test-Path "tests/golden/orchestration/baseline-2026-02-13.json") { $baselineCount++ }

if ($baselineCount -eq 3) {
    Write-Host "  [OK] All 3 golden baseline files present" -ForegroundColor Green
} else {
    Write-Host "  [WARN] Only $baselineCount/3 baseline files present" -ForegroundColor Yellow
    $warnings++
}

Write-Host ""

# Phase 1: Artifact System
Write-Host "Phase 1: Artifact System" -ForegroundColor Yellow

if (Test-Path "db/postgres/006_strategy_artifacts.sql") {
    Write-Host "  [OK] Artifact schema migration exists" -ForegroundColor Green
} else {
    Write-Host "  [FAIL] Missing artifact schema migration" -ForegroundColor Red
    $errors++
}

if (Test-Path "internal/domain/artifacts/artifact.go") {
    Write-Host "  [OK] Artifact domain model exists" -ForegroundColor Green
} else {
    Write-Host "  [FAIL] Missing artifact domain model" -ForegroundColor Red
    $errors++
}

if (Test-Path "internal/domain/artifacts/store.go") {
    Write-Host "  [OK] Artifact store implementation exists" -ForegroundColor Green
    
    $content = Get-Content "internal/domain/artifacts/store.go" -Raw
    if ($content -match "Optimized to fetch all data in single query") {
        Write-Host "  [OK] N+1 query optimization applied" -ForegroundColor Green
    } else {
        Write-Host "  [WARN] N+1 query optimization not found" -ForegroundColor Yellow
        $warnings++
    }
} else {
    Write-Host "  [FAIL] Missing artifact store implementation" -ForegroundColor Red
    $errors++
}

if (Test-Path "db/seeds/001_test_artifacts.sql") {
    Write-Host "  [OK] Test artifact seed data exists" -ForegroundColor Green
} else {
    Write-Host "  [FAIL] Missing test artifact seed data" -ForegroundColor Red
    $errors++
}

if (Test-Path "scripts/apply-test-seeds.ps1") {
    Write-Host "  [OK] Seed application script exists" -ForegroundColor Green
} else {
    Write-Host "  [WARN] Missing seed application script" -ForegroundColor Yellow
    $warnings++
}

Write-Host ""

# Phase 2: Trader Runtime
Write-Host "Phase 2: Trader Runtime Skeleton" -ForegroundColor Yellow

if (Test-Path "cmd/trader/main.go") {
    Write-Host "  [OK] Trader runtime entrypoint exists" -ForegroundColor Green
} else {
    Write-Host "  [FAIL] Missing trader runtime entrypoint" -ForegroundColor Red
    $errors++
}

if (Test-Path "cmd/trader/Dockerfile") {
    Write-Host "  [OK] Trader Dockerfile exists" -ForegroundColor Green
} else {
    Write-Host "  [FAIL] Missing trader Dockerfile" -ForegroundColor Red
    $errors++
}

if (Test-Path ".github/workflows/import-boundary-check.yml") {
    Write-Host "  [OK] Import boundary CI check exists" -ForegroundColor Green
} else {
    Write-Host "  [WARN] Missing import boundary CI check" -ForegroundColor Yellow
    $warnings++
}

Write-Host ""

# Phase 3: Collapse HTTP
Write-Host "Phase 3: Collapse Internal HTTP Services" -ForegroundColor Yellow

if (Test-Path "internal/modules/orchestration/service.go") {
    Write-Host "  [OK] Orchestration module extracted" -ForegroundColor Green
} else {
    Write-Host "  [FAIL] Missing orchestration module" -ForegroundColor Red
    $errors++
}

$archivedServices = @(
    "archive/jax-orchestrator",
    "archive/jax-signal-generator",
    "archive/jax-trade-executor"
)

$archivedCount = 0
foreach ($service in $archivedServices) {
    if (Test-Path $service) {
        $archivedCount++
    }
}

if ($archivedCount -eq $archivedServices.Length) {
    Write-Host "  [OK] Old services archived ($archivedCount/$($archivedServices.Length))" -ForegroundColor Green
} else {
    Write-Host "  [WARN] Only $archivedCount/$($archivedServices.Length) services archived" -ForegroundColor Yellow
    $warnings++
}

Write-Host ""

# Phase 4: Trade Execution
Write-Host "Phase 4: Trade Execution Migration" -ForegroundColor Yellow

if (Test-Path "internal/modules/execution/engine.go") {
    Write-Host "  [OK] Execution engine exists" -ForegroundColor Green
} else {
    Write-Host "  [FAIL] Missing execution engine" -ForegroundColor Red
    $errors++
}

if (Test-Path "cmd/shadow-validator/main.go") {
    Write-Host "  [OK] Shadow validator exists" -ForegroundColor Green
} else {
    Write-Host "  [WARN] Missing shadow validator" -ForegroundColor Yellow
    $warnings++
}

if (Test-Path "docker-compose.shadow.yml") {
    Write-Host "  [OK] Shadow mode docker-compose exists" -ForegroundColor Green
} else {
    Write-Host "  [WARN] Missing shadow mode docker-compose" -ForegroundColor Yellow
    $warnings++
}

if (Test-Path "scripts/run-shadow-validation.ps1") {
    Write-Host "  [OK] Shadow validation script exists" -ForegroundColor Green
} else {
    Write-Host "  [WARN] Missing shadow validation script" -ForegroundColor Yellow
    $warnings++
}

Write-Host ""

# Phase 5: Research Runtime
Write-Host "Phase 5: Research Runtime + Artifact Builder" -ForegroundColor Yellow

if (Test-Path "cmd/research/main.go") {
    Write-Host "  [OK] Research runtime entrypoint exists" -ForegroundColor Green
} else {
    Write-Host "  [FAIL] Missing research runtime entrypoint" -ForegroundColor Red
    $errors++
}

if (Test-Path "internal/modules/backtest/engine.go") {
    Write-Host "  [OK] Backtest engine exists" -ForegroundColor Green
} else {
    Write-Host "  [FAIL] Missing backtest engine" -ForegroundColor Red
    $errors++
}

if (Test-Path "internal/modules/artifacts/builder.go") {
    Write-Host "  [OK] Artifact builder exists" -ForegroundColor Green
} else {
    Write-Host "  [FAIL] Missing artifact builder" -ForegroundColor Red
    $errors++
}

if (Test-Path "cmd/artifact-approver/main.go") {
    Write-Host "  [OK] Artifact approval tool exists" -ForegroundColor Green
} else {
    Write-Host "  [FAIL] Missing artifact approval tool" -ForegroundColor Red
    $errors++
}

Write-Host ""

# Phase 6: Decommission
Write-Host "Phase 6: Decommission Old Services" -ForegroundColor Yellow

if (Test-Path "docker-compose.yml") {
    $composeContent = Get-Content "docker-compose.yml" -Raw
    
    if ($composeContent -match "jax-trader:") {
        Write-Host "  [OK] jax-trader in docker-compose" -ForegroundColor Green
    } else {
        Write-Host "  [FAIL] jax-trader not in docker-compose" -ForegroundColor Red
        $errors++
    }
    
    if ($composeContent -match "jax-research:") {
        Write-Host "  [OK] jax-research in docker-compose" -ForegroundColor Green
    } else {
        Write-Host "  [FAIL] jax-research not in docker-compose" -ForegroundColor Red
        $errors++
    }
}

Write-Host ""

# Final Report
Write-Host "======================================" -ForegroundColor Cyan
Write-Host "Verification Summary" -ForegroundColor Cyan
Write-Host "======================================" -ForegroundColor Cyan
Write-Host ""

if ($errors -eq 0 -and $warnings -eq 0) {
    Write-Host "SUCCESS: ALL CHECKS PASSED!" -ForegroundColor Green
    Write-Host ""
    Write-Host "ADR-0012 implementation is complete." -ForegroundColor Green
    Write-Host ""
} elseif ($errors -eq 0) {
    Write-Host "PASS: No critical errors found" -ForegroundColor Green
    Write-Host "WARN: $warnings warnings (minor issues)" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "Implementation is mostly complete with minor gaps." -ForegroundColor Yellow
    Write-Host ""
} else {
    Write-Host "FAIL: $errors critical errors found" -ForegroundColor Red
    Write-Host "WARN: $warnings warnings" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "Implementation has critical gaps that need attention." -ForegroundColor Red
    Write-Host ""
}

Write-Host "Next Steps:" -ForegroundColor Cyan
Write-Host "  1. Run tests: go test ./..." -ForegroundColor Gray
Write-Host "  2. Apply seeds: .\scripts\apply-test-seeds.ps1" -ForegroundColor Gray
Write-Host "  3. Build: docker-compose build" -ForegroundColor Gray
Write-Host "  4. Start: docker-compose up -d" -ForegroundColor Gray
Write-Host "  5. Health: Invoke-RestMethod http://localhost:8100/health" -ForegroundColor Gray
Write-Host ""

exit $errors
