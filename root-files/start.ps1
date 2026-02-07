# JAX Trading Assistant - Start Script
# Starts all services and opens the dashboard

$ErrorActionPreference = "Continue"

Write-Host "Starting JAX Trading Assistant..." -ForegroundColor Green
Write-Host "See Docs/DEBUGGING.md for troubleshooting" -ForegroundColor Gray

# Check if .env exists
if (-not (Test-Path ".env")) {
    Write-Host "Creating .env from .env.example..." -ForegroundColor Yellow
    Copy-Item ".env.example" ".env"
}

# Start backend services
Write-Host "`nStarting backend services (Docker)..." -ForegroundColor Cyan

# Start postgres first
Write-Host "  Starting postgres..." -ForegroundColor Gray
docker compose -f root-files/docker-compose.yml up -d postgres 2>$null
Start-Sleep -Seconds 3

# Wait for postgres to be healthy
for ($i = 1; $i -le 10; $i++) {
    $pgStatus = docker compose -f root-files/docker-compose.yml ps postgres --format json 2>$null | ConvertFrom-Json
    if ($pgStatus.Health -eq "healthy") {
        Write-Host "  Postgres is ready" -ForegroundColor Green
        break
    }
    Write-Host "  Waiting for postgres... ($i/10)" -ForegroundColor Gray
    Start-Sleep -Seconds 2
}

# Start other services
Write-Host "  Starting hindsight, jax-memory, jax-api..." -ForegroundColor Gray
docker compose -f root-files/docker-compose.yml up -d hindsight jax-memory jax-api 2>$null

# Wait for services to be ready
Write-Host "`nWaiting for services to be ready..." -ForegroundColor Yellow
Start-Sleep -Seconds 10

# Check health
$apiHealthy = $false
$memoryHealthy = $false

for ($i = 1; $i -le 6; $i++) {
    try {
        $apiResponse = Invoke-WebRequest -Uri "http://localhost:8081/health" -TimeoutSec 2 -UseBasicParsing -ErrorAction SilentlyContinue
        if ($apiResponse.StatusCode -eq 200) { $apiHealthy = $true }
    } catch { }

    try {
        $memoryResponse = Invoke-WebRequest -Uri "http://localhost:8090/health" -TimeoutSec 2 -UseBasicParsing -ErrorAction SilentlyContinue
        if ($memoryResponse.StatusCode -eq 200) { $memoryHealthy = $true }
    } catch { }

    if ($apiHealthy -and $memoryHealthy) {
        Write-Host "Backend services are ready!" -ForegroundColor Green
        break
    }

    if ($i -lt 6) {
        Write-Host "   Waiting... ($i/6)" -ForegroundColor Gray
        Start-Sleep -Seconds 5
    }
}

if (-not ($apiHealthy -and $memoryHealthy)) {
    Write-Host "Backend services may not be fully ready, continuing anyway..." -ForegroundColor Yellow
}

# Start frontend
Write-Host "`nStarting frontend (React)..." -ForegroundColor Cyan
Push-Location frontend

# Check if node_modules exists
if (-not (Test-Path "node_modules")) {
    Write-Host "Installing dependencies..." -ForegroundColor Yellow
    npm install
}

# Open browser after a short delay
Start-Job -ScriptBlock {
    Start-Sleep -Seconds 3
    Start-Process "http://localhost:5173"
} | Out-Null

Write-Host "`nOpening dashboard at http://localhost:5173" -ForegroundColor Green
Write-Host "Press Ctrl+C to stop`n" -ForegroundColor Gray

# Start dev server (this will keep running)
npm run dev

Pop-Location
