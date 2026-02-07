# JAX Trading Assistant - Stop Script
# Stops all services

Write-Host "🛑 Stopping JAX Trading Assistant..." -ForegroundColor Yellow

# Stop docker services
Write-Host "`n📦 Stopping backend services..." -ForegroundColor Cyan
docker compose down

Write-Host "`n✅ All services stopped" -ForegroundColor Green
Write-Host "Note: Frontend dev server (if running) should be stopped with Ctrl+C" -ForegroundColor Gray
