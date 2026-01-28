# JAX Trading Assistant - Start Script
# Starts all services and opens the dashboard

Write-Host "🚀 Starting JAX Trading Assistant..." -ForegroundColor Green

# Start backend services
Write-Host "`n📦 Starting backend services (Docker)..." -ForegroundColor Cyan
docker compose up -d

if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Failed to start backend services" -ForegroundColor Red
    exit 1
}

# Wait for services to be ready
Write-Host "`n⏳ Waiting for services to be ready..." -ForegroundColor Yellow
Start-Sleep -Seconds 10

# Check health
$apiHealthy = $false
$memoryHealthy = $false

for ($i = 1; $i -le 6; $i++) {
    try {
        $apiResponse = Invoke-WebRequest -Uri "http://localhost:8081/health" -TimeoutSec 2 -UseBasicParsing -ErrorAction SilentlyContinue
        if ($apiResponse.StatusCode -eq 200) { $apiHealthy = $true }
    } catch {}
    
    try {
        $memoryResponse = Invoke-WebRequest -Uri "http://localhost:8090/health" -TimeoutSec 2 -UseBasicParsing -ErrorAction SilentlyContinue
        if ($memoryResponse.StatusCode -eq 200) { $memoryHealthy = $true }
    } catch {}
    
    if ($apiHealthy -and $memoryHealthy) {
        Write-Host "✅ Backend services are ready!" -ForegroundColor Green
        break
    }
    
    if ($i -lt 6) {
        Write-Host "   Waiting... ($i/6)" -ForegroundColor Gray
        Start-Sleep -Seconds 5
    }
}

if (-not ($apiHealthy -and $memoryHealthy)) {
    Write-Host "⚠️  Backend services may not be fully ready, continuing anyway..." -ForegroundColor Yellow
}

# Start frontend
Write-Host "`n🎨 Starting frontend (React)..." -ForegroundColor Cyan
Push-Location frontend

# Check if node_modules exists
if (-not (Test-Path "node_modules")) {
    Write-Host "📦 Installing dependencies..." -ForegroundColor Yellow
    npm install
}

# Open browser after a short delay
Start-Job -ScriptBlock {
    Start-Sleep -Seconds 3
    Start-Process "http://localhost:5173"
} | Out-Null

Write-Host "`n✨ Opening dashboard at http://localhost:5173" -ForegroundColor Green
Write-Host "Press Ctrl+C to stop`n" -ForegroundColor Gray

# Start dev server (this will keep running)
npm run dev

Pop-Location
