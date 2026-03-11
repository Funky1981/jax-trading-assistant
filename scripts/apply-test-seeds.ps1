# Apply approved strategy artifact seeds using the running Postgres container.

Write-Host "Applying test artifact seeds..." -ForegroundColor Cyan

$repoRoot = Split-Path -Parent $PSScriptRoot
$seedFile = Join-Path (Join-Path (Join-Path $repoRoot "db") "seeds") "001_test_artifacts.sql"

if (-not (Test-Path $seedFile)) {
    Write-Host "Seed file not found: $seedFile" -ForegroundColor Red
    exit 1
}

Write-Host "Seed file: $seedFile" -ForegroundColor Gray

$postgresContainer = "jax-tradingassistant-postgres-1"
$containerSeedPath = "/tmp/001_test_artifacts.sql"

try {
    docker cp $seedFile "${postgresContainer}:${containerSeedPath}" | Out-Null
    if ($LASTEXITCODE -ne 0) {
        throw "docker cp failed"
    }

    docker compose exec -T postgres psql -U jax -d jax -f $containerSeedPath
    if ($LASTEXITCODE -ne 0) {
        throw "psql seed apply failed"
    }

    Write-Host ""
    Write-Host "Test artifacts seeded successfully" -ForegroundColor Green
    Write-Host ""
    Write-Host "Available approved artifacts:" -ForegroundColor Cyan

    $query = "SELECT sa.artifact_id, sa.strategy_name, aa.state, aa.validation_passed FROM strategy_artifacts sa JOIN artifact_approvals aa ON sa.id = aa.artifact_id WHERE aa.state IN ('APPROVED','ACTIVE') ORDER BY sa.created_at DESC;"
    docker compose exec -T postgres psql -U jax -d jax -c $query
}
catch {
    Write-Host "Error applying seeds: $_" -ForegroundColor Red
    exit 1
}
