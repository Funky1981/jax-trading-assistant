# JAX Trading Assistant - Start Script
# Starts all services and opens the dashboard

$ErrorActionPreference = "Continue"
$RuntimeDir = ".runtime"
$FrontendPidFile = Join-Path $RuntimeDir "frontend-dev.pid"
$FrontendLogFile = "logs/frontend-dev.log"

function Ensure-Directory([string]$Path) {
    if (-not (Test-Path $Path)) {
        New-Item -ItemType Directory -Path $Path | Out-Null
    }
}

function Stop-StaleFrontendProcess {
    if (-not (Test-Path $FrontendPidFile)) {
        return
    }

    try {
        $pid = [int](Get-Content $FrontendPidFile -ErrorAction Stop | Select-Object -First 1)
        $proc = Get-Process -Id $pid -ErrorAction SilentlyContinue
        if ($proc) {
            Write-Host "  Stopping previous frontend dev server (PID $pid)..." -ForegroundColor Gray
            Stop-Process -Id $pid -Force -ErrorAction SilentlyContinue
            Start-Sleep -Seconds 1
        }
    } catch { }

    Remove-Item $FrontendPidFile -ErrorAction SilentlyContinue
}

function Wait-ForHttp([string]$Url, [int]$Attempts = 20, [int]$DelaySeconds = 2) {
    for ($i = 1; $i -le $Attempts; $i++) {
        try {
            $response = Invoke-WebRequest -Uri $Url -TimeoutSec 2 -UseBasicParsing -ErrorAction Stop
            if ($response.StatusCode -ge 200 -and $response.StatusCode -lt 500) {
                return $true
            }
        } catch { }

        if ($i -lt $Attempts) {
            Start-Sleep -Seconds $DelaySeconds
        }
    }

    return $false
}

Write-Host "Starting JAX Trading Assistant..." -ForegroundColor Green
Write-Host "See Docs/DEBUGGING.md for troubleshooting" -ForegroundColor Gray

Ensure-Directory $RuntimeDir
Ensure-Directory "logs"

# Check if .env exists
if (-not (Test-Path ".env")) {
    Write-Host "Creating .env from .env.example..." -ForegroundColor Yellow
    Copy-Item ".env.example" ".env"
}

# Start backend services
Write-Host "`nStarting backend services (Docker)..." -ForegroundColor Cyan

# Force container-friendly DATABASE_URL unless explicitly set in this session.
if (-not $env:DATABASE_URL) {
    $env:DATABASE_URL = "postgresql://jax:jax@postgres:5432/jax"
}

# Build latest service images unless explicitly skipped.
if ($env:JAX_SKIP_BUILD -ne "true") {
    Write-Host "  Building jax-trader and jax-research..." -ForegroundColor Gray
    docker compose build jax-trader jax-research 2>$null
}

# Start postgres first
Write-Host "  Starting postgres..." -ForegroundColor Gray
docker compose up -d postgres 2>$null
Start-Sleep -Seconds 3

# Wait for postgres to be healthy
for ($i = 1; $i -le 10; $i++) {
    $pgStatus = docker compose ps postgres --format json 2>$null | ConvertFrom-Json
    if ($pgStatus.Health -eq "healthy") {
        Write-Host "  Postgres is ready" -ForegroundColor Green
        break
    }
    Write-Host "  Waiting for postgres... ($i/10)" -ForegroundColor Gray
    Start-Sleep -Seconds 2
}

# Start other services
Write-Host "  Starting core services..." -ForegroundColor Gray
docker compose up -d --force-recreate jax-trader jax-research ib-bridge agent0-service hindsight prometheus grafana 2>$null

# Wait for services to be ready
Write-Host "`nWaiting for services to be ready..." -ForegroundColor Yellow
Start-Sleep -Seconds 10

# Check health
$apiHealthy = $false
$researchHealthy = $false
$bridgeHealthy = $false
$agentHealthy = $false

for ($i = 1; $i -le 6; $i++) {
    try {
        $apiResponse = Invoke-WebRequest -Uri "http://localhost:8081/health" -TimeoutSec 2 -UseBasicParsing -ErrorAction SilentlyContinue
        if ($apiResponse.StatusCode -eq 200) { $apiHealthy = $true }
    } catch { }

    try {
        $researchResponse = Invoke-WebRequest -Uri "http://localhost:8091/health" -TimeoutSec 2 -UseBasicParsing -ErrorAction SilentlyContinue
        if ($researchResponse.StatusCode -eq 200) { $researchHealthy = $true }
    } catch { }

    try {
        $bridgeResponse = Invoke-WebRequest -Uri "http://localhost:8092/health" -TimeoutSec 2 -UseBasicParsing -ErrorAction SilentlyContinue
        if ($bridgeResponse.StatusCode -eq 200) { $bridgeHealthy = $true }
    } catch { }

    try {
        $agentResponse = Invoke-WebRequest -Uri "http://localhost:8093/health" -TimeoutSec 2 -UseBasicParsing -ErrorAction SilentlyContinue
        if ($agentResponse.StatusCode -eq 200) { $agentHealthy = $true }
    } catch { }

    if ($apiHealthy -and $researchHealthy -and $bridgeHealthy -and $agentHealthy) {
        Write-Host "Backend services are ready!" -ForegroundColor Green
        break
    }

    if ($i -lt 6) {
        Write-Host "   Waiting... ($i/6)" -ForegroundColor Gray
        Start-Sleep -Seconds 5
    }
}

if (-not ($apiHealthy -and $researchHealthy -and $bridgeHealthy -and $agentHealthy)) {
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

Pop-Location

Stop-StaleFrontendProcess
Write-Host "  Launching frontend dev server..." -ForegroundColor Gray
$frontendProcess = Start-Process powershell `
    -ArgumentList '-NoProfile', '-Command', "Set-Location '$PWD\frontend'; npm run dev *> '$PWD\$FrontendLogFile'" `
    -PassThru `
    -WindowStyle Hidden
$frontendProcess.Id | Set-Content $FrontendPidFile

if (-not (Wait-ForHttp "http://localhost:5173/" 30 1)) {
    Write-Host "Frontend failed to become ready on http://localhost:5173" -ForegroundColor Red
    Write-Host "Last frontend log lines:" -ForegroundColor Yellow
    if (Test-Path $FrontendLogFile) {
        Get-Content $FrontendLogFile -Tail 40
    }
    exit 1
}

Write-Host "`nOpening dashboard at http://localhost:5173" -ForegroundColor Green
Start-Process "http://localhost:5173" | Out-Null
Write-Host "Frontend dev server PID: $($frontendProcess.Id)" -ForegroundColor Gray
Write-Host "Frontend log: $FrontendLogFile" -ForegroundColor Gray
Write-Host "Use .\\stop.ps1 to stop backend and frontend`n" -ForegroundColor Gray
