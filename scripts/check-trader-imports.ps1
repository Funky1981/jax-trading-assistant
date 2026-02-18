#!/usr/bin/env pwsh
# scripts/check-trader-imports.ps1
#
# ADR-0012 Phase 5: Import boundary guard for cmd/trader.
# The Trader runtime must NOT import research-only libraries.
# Run this in CI after every change to cmd/trader or its transitive deps.
#
# Usage:
#   ./scripts/check-trader-imports.ps1
#   exit code 0 = clean, exit code 1 = violation found

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot

Write-Host "=== Trader import boundary check ===" -ForegroundColor Cyan
Write-Host "Scanning: cmd/trader" -ForegroundColor Gray

# Packages that must NEVER appear in cmd/trader's dependency tree
$denied = @(
    "jax-trading-assistant/libs/dexter",
    "jax-trading-assistant/libs/agent0",
    "jax-trading-assistant/services/hindsight",
    "jax-trading-assistant/internal/modules/orchestration"  # orchestration is research-domain
)

Push-Location $root
try {
    $deps = go list -deps ./cmd/trader 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Host "ERROR: go list failed:" -ForegroundColor Red
        Write-Host $deps
        exit 1
    }

    $violations = @()
    foreach ($denied_pkg in $denied) {
        $found = $deps | Where-Object { $_ -like "*$denied_pkg*" }
        if ($found) {
            $violations += $denied_pkg
        }
    }

    if ($violations.Count -gt 0) {
        Write-Host ""
        Write-Host "BOUNDARY VIOLATION: cmd/trader imports research-only packages:" -ForegroundColor Red
        foreach ($v in $violations) {
            Write-Host "  [FAIL] $v" -ForegroundColor Red
        }
        Write-Host ""
        Write-Host "Fix: move research-specific logic to cmd/research." -ForegroundColor Yellow
        exit 1
    }

    Write-Host "[OK] No research-only packages in cmd/trader dependency tree." -ForegroundColor Green
    exit 0
} finally {
    Pop-Location
}
