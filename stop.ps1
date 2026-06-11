# JAX Trading Assistant - Stop Script
# Stops frontend and backend services

$RuntimeDir = ".runtime"
$FrontendPidFile = Join-Path $RuntimeDir "frontend-dev.pid"
$AgentPidFile = Join-Path $RuntimeDir "playwright-agent.pid"

function Stop-ListeningProcessOnPort([int]$Port) {
    $matches = netstat -ano 2>$null | Select-String ":$Port\s+.*LISTENING"
    foreach ($match in $matches) {
        $pidText = ($match.Line.Trim() -split '\s+')[-1]
        if ($pidText -match '^\d+$') {
            $portPid = [int]$pidText
            $proc = Get-Process -Id $portPid -ErrorAction SilentlyContinue
            if ($proc) {
                Write-Host "Stopping stale frontend listener on port $Port (PID $portPid)..." -ForegroundColor Cyan
                Stop-Process -Id $portPid -Force -ErrorAction SilentlyContinue
            }
        }
    }
}

Write-Host "Stopping JAX Trading Assistant..." -ForegroundColor Yellow

if (Test-Path $FrontendPidFile) {
    try {
        $pid = [int](Get-Content $FrontendPidFile -ErrorAction Stop | Select-Object -First 1)
        $proc = Get-Process -Id $pid -ErrorAction SilentlyContinue
        if ($proc) {
            Write-Host "`nStopping frontend dev server (PID $pid)..." -ForegroundColor Cyan
            Stop-Process -Id $pid -Force -ErrorAction SilentlyContinue
        }
    } catch { }

    Remove-Item $FrontendPidFile -ErrorAction SilentlyContinue
}

Stop-ListeningProcessOnPort 5173
Stop-ListeningProcessOnPort 5174

if (Test-Path $AgentPidFile) {
    try {
        $agentPid = [int](Get-Content $AgentPidFile -ErrorAction Stop | Select-Object -First 1)
        $agentProc = Get-Process -Id $agentPid -ErrorAction SilentlyContinue
        if ($agentProc) {
            Write-Host "`nStopping Playwright test agent (PID $agentPid)..." -ForegroundColor Cyan
            Stop-Process -Id $agentPid -Force -ErrorAction SilentlyContinue
        }
    } catch { }
    Remove-Item $AgentPidFile -ErrorAction SilentlyContinue
}

Write-Host "`nStopping backend services..." -ForegroundColor Cyan
docker compose down

Write-Host "`nAll services stopped" -ForegroundColor Green
