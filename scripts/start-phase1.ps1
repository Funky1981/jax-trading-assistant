# Phase 1 Quick Start Script
# This script starts only the services needed for Phase 1 testing

Write-Host "🚀 Starting Phase 1 Services..." -ForegroundColor Cyan

# Stop any running services first
Write-Host "`n📋 Stopping existing services..." -ForegroundColor Yellow
docker compose down

# Start core dependencies first
Write-Host "`n📦 Starting core infrastructure..." -ForegroundColor Yellow
docker compose up -d postgres
Start-Sleep -Seconds 10  # Wait for postgres to be ready

# Check postgres health
Write-Host "`n🔍 Checking PostgreSQL..." -ForegroundColor Yellow
docker compose ps postgres

# Start Phase 1 services
Write-Host "`n🏗️  Building and starting jax-market..." -ForegroundColor Yellow
docker compose up -d --build jax-market

# Wait for service to start
Start-Sleep -Seconds 15

# Check service status
Write-Host "`n📊 Service Status:" -ForegroundColor Cyan
docker compose ps

# Test health endpoint
Write-Host "`n🏥 Testing jax-market health..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "http://localhost:8095/health" -TimeoutSec 5
    Write-Host "✅ Health check PASSED" -ForegroundColor Green
    Write-Host $response.Content
} catch {
    Write-Host "❌ Health check FAILED" -ForegroundColor Red
    Write-Host "Service may still be starting. Wait 30 seconds and try:"
    Write-Host "curl http://localhost:8095/health" -ForegroundColor Yellow
}

# Test metrics endpoint
Write-Host "`n📈 Testing jax-market metrics..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "http://localhost:8095/metrics" -TimeoutSec 5
    Write-Host "✅ Metrics endpoint PASSED" -ForegroundColor Green
    Write-Host $response.Content
} catch {
    Write-Host "❌ Metrics endpoint FAILED" -ForegroundColor Red
}

# Show logs
Write-Host "`n📜 Recent jax-market logs:" -ForegroundColor Cyan
docker compose logs --tail=20 jax-market

Write-Host "`n✨ Phase 1 startup complete!" -ForegroundColor Green
Write-Host "`nNext steps:" -ForegroundColor Cyan
Write-Host "  1. Check logs: docker compose logs -f jax-market"
Write-Host "  2. View metrics: curl http://localhost:8095/metrics"
Write-Host "  3. Query database: docker compose exec postgres psql -U jax -d jax -c 'SELECT * FROM quotes LIMIT 5;'"
Write-Host "`nFor full testing guide, see: PHASE_1_TESTING.md" -ForegroundColor Gray
