# Apply Test Artifact Seeds
# ADR-0012 Phase 1: Provides approved test artifacts for local development

Write-Host "🌱 Applying test artifact seeds..." -ForegroundColor Cyan

# Check if DATABASE_URL is set
if (-not $env:DATABASE_URL) {
    Write-Host "⚠️  DATABASE_URL not set. Using default..." -ForegroundColor Yellow
    $env:DATABASE_URL = "postgresql://jax:password@localhost:5433/jax"
}

Write-Host "Database: $env:DATABASE_URL" -ForegroundColor Gray

# Apply seed file
$seedFile = Join-Path $PSScriptRoot ".." "db" "seeds" "001_test_artifacts.sql"

if (-not (Test-Path $seedFile)) {
    Write-Host "❌ Seed file not found: $seedFile" -ForegroundColor Red
    exit 1
}

Write-Host "Applying seed: $seedFile" -ForegroundColor Gray

# Use psql to apply the seed
try {
    $env:PGPASSWORD = "password"
    psql $env:DATABASE_URL -f $seedFile 2>&1 | Out-String | Write-Host
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host ""
        Write-Host "✅ Test artifacts seeded successfully" -ForegroundColor Green
        Write-Host ""
        Write-Host "Available test artifacts:" -ForegroundColor Cyan
        
        # Query and display seeded artifacts
        $query = "SELECT artifact_id, strategy_name, aa.state FROM strategy_artifacts sa JOIN artifact_approvals aa ON sa.id = aa.artifact_id WHERE created_by = 'seed'"
        psql $env:DATABASE_URL -c $query 2>&1 | Out-String | Write-Host
    }
    else {
        Write-Host "❌ Failed to apply seeds" -ForegroundColor Red
        exit 1
    }
}
catch {
    Write-Host "❌ Error applying seeds: $_" -ForegroundColor Red
    exit 1
}
finally {
    Remove-Item Env:\PGPASSWORD -ErrorAction SilentlyContinue
}
