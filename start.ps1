# JAX Trading Assistant - Start Script
# Starts all services and opens the dashboard

$ErrorActionPreference = "Continue"
$RuntimeDir = ".runtime"
$FrontendPidFile = Join-Path $RuntimeDir "frontend-dev.pid"
$FrontendLogFile = "logs/frontend-dev.log"

function Initialize-Directory([string]$Path) {
    if (-not (Test-Path $Path)) {
        New-Item -ItemType Directory -Path $Path | Out-Null
    }
}

function Stop-StaleFrontendProcess {
    if (-not (Test-Path $FrontendPidFile)) {
        return
    }

    try {
        $procID = [int](Get-Content $FrontendPidFile -ErrorAction Stop | Select-Object -First 1)
        $proc = Get-Process -Id $procID -ErrorAction SilentlyContinue
        if ($proc) {
            Write-Host "  Stopping previous frontend dev server (PID $procID)..." -ForegroundColor Gray
            Stop-Process -Id $procID -Force -ErrorAction SilentlyContinue
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

Initialize-Directory $RuntimeDir
Initialize-Directory "logs"

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
if (-not $env:EMBEDDING_PROVIDER) {
    $env:EMBEDDING_PROVIDER = "local"
}

# Build service images: auto when any required image is missing, or when JAX_BUILD=true.
$requiredImages = @(
    "jax-trading-assistant-db-migrate",
    "jax-trading-assistant-jax-trader",
    "jax-trading-assistant-jax-research",
    "jax-trading-assistant-ib-bridge",
    "jax-trading-assistant-agent0-service"
)
$needsBuild = $env:JAX_BUILD -eq "true"
if (-not $needsBuild) {
    $missing = $requiredImages | Where-Object {
        -not (docker image inspect "${_}:latest" --format "{{.Id}}" 2>$null)
    }
    if ($missing) {
        Write-Host "  Missing images: $($missing -join ', ') - building..." -ForegroundColor Yellow
        $needsBuild = $true
    } else {
        Write-Host "  All images cached. Skipping build (set JAX_BUILD=true to force rebuild)." -ForegroundColor Gray
    }
}
if ($needsBuild) {
    Write-Host "  Building service images..." -ForegroundColor Gray
    docker compose build 2>$null
}

# Start postgres first
Write-Host "  Starting postgres..." -ForegroundColor Gray
docker compose up -d --no-build postgres 2>$null
Start-Sleep -Seconds 3

# Wait for postgres to be healthy
for ($i = 1; $i -le 10; $i++) {
    $pgJson = docker compose ps postgres --format json 2>$null
    try {
        $pgStatus = $pgJson | ConvertFrom-Json
        # Docker Compose may return an array or a single object
        $health = if ($pgStatus -is [array]) { $pgStatus[0].Health } else { $pgStatus.Health }
        if ($health -eq "healthy") {
            Write-Host "  Postgres is ready" -ForegroundColor Green
            break
        }
    } catch { }
    Write-Host "  Waiting for postgres... ($i/10)" -ForegroundColor Gray
    Start-Sleep -Seconds 2
}

# Run any pending migrations via the db-migrate service so schema_migrations stays correct.
Write-Host "  Applying database migrations..." -ForegroundColor Gray
docker compose rm -f db-migrate 2>$null | Out-Null
docker compose up --no-build db-migrate 2>$null
if ($LASTEXITCODE -ne 0) {
    Write-Host "  Migration step failed." -ForegroundColor Red
    exit $LASTEXITCODE
}
Write-Host "  Migrations applied." -ForegroundColor Green

# Start other services
Write-Host "  Starting core services..." -ForegroundColor Gray
docker compose up -d --no-build jax-trader jax-research ib-bridge agent0-service prometheus grafana 2>$null

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
        $apiResponse = Invoke-WebRequest -Uri "http://localhost:8100/ready" -TimeoutSec 2 -UseBasicParsing -ErrorAction SilentlyContinue
        if ($apiResponse.StatusCode -eq 200) { $apiHealthy = $true }
    } catch { }

    try {
        $researchResponse = Invoke-WebRequest -Uri "http://localhost:8091/ready" -TimeoutSec 2 -UseBasicParsing -ErrorAction SilentlyContinue
        if ($researchResponse.StatusCode -eq 200) { $researchHealthy = $true }
    } catch { }

    try {
        $bridgeResponse = Invoke-WebRequest -Uri "http://localhost:8092/ready" -TimeoutSec 2 -UseBasicParsing -ErrorAction SilentlyContinue
        if ($bridgeResponse.StatusCode -eq 200) { $bridgeHealthy = $true }
    } catch { }

    try {
        $agentResponse = Invoke-WebRequest -Uri "http://localhost:8093/ready" -TimeoutSec 2 -UseBasicParsing -ErrorAction SilentlyContinue
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

# Kill any orphaned process still holding port 5173 (e.g. from a previous crashed run).
$netout = netstat -ano 2>$null | Select-String ":5173\s.*LISTENING"
if ($netout) {
    $stalePid = ($netout.ToString().Trim() -split '\s+')[-1]
    if ($stalePid -match '^\d+$') {
        Write-Host "  Killing stale process on port 5173 (PID $stalePid)..." -ForegroundColor Gray
        Stop-Process -Id ([int]$stalePid) -Force -ErrorAction SilentlyContinue
        Start-Sleep -Milliseconds 500
    }
}

Write-Host "  Launching frontend dev server..." -ForegroundColor Gray
$frontendProcess = Start-Process powershell `
    -ArgumentList '-NoProfile', '-Command', "Set-Location '$PWD\frontend'; npm run dev *> '$PWD\$FrontendLogFile'" `
    -PassThru `
    -WindowStyle Hidden
$frontendProcess.Id | Set-Content $FrontendPidFile

if (-not (Wait-ForHttp "http://localhost:5173/" 60 2)) {
    Write-Host "Frontend failed to become ready on http://localhost:5173" -ForegroundColor Red
    Write-Host "Last frontend log lines:" -ForegroundColor Yellow
    if (Test-Path $FrontendLogFile) {
        Get-Content $FrontendLogFile -Tail 40
    }
    exit 1
}

# Start the Playwright test agent (host-side HTTP runner for E2E tests in the container)
$AgentPidFile = Join-Path $RuntimeDir "playwright-agent.pid"
$AgentLogFile = "logs/playwright-agent.log"

# Kill stale agent if running
if (Test-Path $AgentPidFile) {
    try {
        $stalePid = [int](Get-Content $AgentPidFile -ErrorAction Stop | Select-Object -First 1)
        $staleProc = Get-Process -Id $stalePid -ErrorAction SilentlyContinue
        if ($staleProc) { Stop-Process -Id $stalePid -Force -ErrorAction SilentlyContinue }
    } catch { }
    Remove-Item $AgentPidFile -ErrorAction SilentlyContinue
}

Write-Host "  Launching Playwright test agent on port 9092..." -ForegroundColor Gray
$AgentScript = Join-Path $PWD "scripts\playwright-agent.js"
$agentProcess = Start-Process node `
    -ArgumentList "`"$AgentScript`"" `
    -RedirectStandardOutput (Join-Path $PWD $AgentLogFile) `
    -RedirectStandardError  (Join-Path $PWD "logs\playwright-agent-err.log") `
    -PassThru `
    -WindowStyle Hidden
$agentProcess.Id | Set-Content $AgentPidFile
Write-Host "  Playwright agent PID: $($agentProcess.Id)" -ForegroundColor Gray

Write-Host "`nOpening dashboard at http://localhost:5173" -ForegroundColor Green
Start-Process "http://localhost:5173" | Out-Null
Write-Host "Frontend dev server PID: $($frontendProcess.Id)" -ForegroundColor Gray
Write-Host "Frontend log: $FrontendLogFile" -ForegroundColor Gray
Write-Host "Use .\\stop.ps1 to stop backend and frontend`n" -ForegroundColor Gray
