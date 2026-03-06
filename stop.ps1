# JAX Trading Assistant - Stop Script
# Stops frontend and backend services

$RuntimeDir = ".runtime"
$FrontendPidFile = Join-Path $RuntimeDir "frontend-dev.pid"

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

Write-Host "`nStopping backend services..." -ForegroundColor Cyan
docker compose down

Write-Host "`nAll services stopped" -ForegroundColor Green
