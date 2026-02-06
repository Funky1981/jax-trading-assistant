# JAX Trading Assistant - Service Status Check
Write-Host "===== JAX Trading Assistant - Service Status =====" -ForegroundColor Cyan
Write-Host ""

# Test PostgreSQL
Write-Host "PostgreSQL: " -NoNewline
try {
    $connection = Test-NetConnection -ComputerName localhost -Port 5432 -WarningAction SilentlyContinue
    if ($connection.TcpTestSucceeded) {
        Write-Host "ONLINE" -ForegroundColor Green
    } else {
        Write-Host "OFFLINE" -ForegroundColor Red
    }
} catch {
    Write-Host "OFFLINE" -ForegroundColor Red
}

# Test Hindsight
Write-Host "Hindsight API: " -NoNewline
try {
    $response = Invoke-RestMethod -Uri "http://localhost:8888/health" -TimeoutSec 2
    Write-Host "ONLINE" -ForegroundColor Green  
} catch {
    Write-Host "OFFLINE - $($_.Exception.Message)" -ForegroundColor Red
}

# Test Memory Service
Write-Host "JAX Memory: " -NoNewline
try {
    $response = Invoke-RestMethod -Uri "http://localhost:8090/health" -TimeoutSec 2
    Write-Host "ONLINE" -ForegroundColor Green
} catch {
    Write-Host "OFFLINE - $($_.Exception.Message)" -ForegroundColor Red
}

# Test IB Bridge
Write-Host "IB Bridge: " -NoNewline
try {
    $response = Invoke-RestMethod -Uri "http://localhost:8092/health" -TimeoutSec 2
    Write-Host "ONLINE" -ForegroundColor Green
    if ($response.connected -ne $null) {
        if ($response.connected) {
            Write-Host "  IB Gateway: Connected" -ForegroundColor Green
        } else {
            Write-Host "  IB Gateway: Disconnected" -ForegroundColor Yellow
        }
    }
} catch {
    Write-Host "OFFLINE - $($_.Exception.Message)" -ForegroundColor Red
}

# Test Agent0
Write-Host "Agent0 AI: " -NoNewline
try {
    $response = Invoke-RestMethod -Uri "http://localhost:8093/health" -TimeoutSec 2
    Write-Host "ONLINE" -ForegroundColor Green
    Write-Host "  Provider: $($response.llm_provider) ($($response.llm_cost))" -ForegroundColor Gray
} catch {
    Write-Host "OFFLINE - $($_.Exception.Message)" -ForegroundColor Red
}

# Test JAX API
Write-Host "JAX API: " -NoNewline
try {
    $response = Invoke-RestMethod -Uri "http://localhost:8081/health" -TimeoutSec 2
    Write-Host "ONLINE" -ForegroundColor Green
} catch {
    Write-Host "OFFLINE - $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host ""
Write-Host "===== IB Gateway Ports =====" -ForegroundColor Cyan
$ports = @(7497, 4001, 4002)
foreach ($port in $ports) {
    Write-Host "Port $port : " -NoNewline
    $test = Test-NetConnection -ComputerName localhost -Port $port -WarningAction SilentlyContinue -InformationLevel Quiet
    if ($test) {
        Write-Host "LISTENING" -ForegroundColor Green
    } else {
        Write-Host "NOT LISTENING" -ForegroundColor Yellow
    }
}

Write-Host ""
