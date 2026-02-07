# Quick Start Script for IB Bridge
# This script helps you get started with the IB Bridge service

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  IB Bridge Quick Start" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Check if Docker is running
Write-Host "Checking Docker..." -ForegroundColor Yellow
try {
    docker ps | Out-Null
    Write-Host "✅ Docker is running" -ForegroundColor Green
} catch {
    Write-Host "❌ Docker is not running. Please start Docker Desktop." -ForegroundColor Red
    exit 1
}

# Check if IB Gateway is accessible
Write-Host ""
Write-Host "Checking IB Gateway..." -ForegroundColor Yellow
$ibPort = 7497  # Paper trading port
$ibHost = "127.0.0.1"

try {
    $tcpClient = New-Object System.Net.Sockets.TcpClient
    $tcpClient.Connect($ibHost, $ibPort)
    $tcpClient.Close()
    Write-Host "✅ IB Gateway is accessible on port $ibPort" -ForegroundColor Green
} catch {
    Write-Host "⚠️  Cannot connect to IB Gateway on port $ibPort" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "Please ensure:" -ForegroundColor Yellow
    Write-Host "  1. IB Gateway is running" -ForegroundColor Yellow
    Write-Host "  2. API access is enabled (Configure → Settings → API)" -ForegroundColor Yellow
    Write-Host "  3. Port is set to 7497 (paper trading)" -ForegroundColor Yellow
    Write-Host ""
    $continue = Read-Host "Continue anyway? (y/n)"
    if ($continue -ne "y") {
        exit 1
    }
}

# Create .env file if it doesn't exist
Write-Host ""
Write-Host "Setting up configuration..." -ForegroundColor Yellow
$envPath = "services\ib-bridge\.env"
if (-not (Test-Path $envPath)) {
    Copy-Item "services\ib-bridge\.env.example" $envPath
    Write-Host "✅ Created .env file" -ForegroundColor Green
} else {
    Write-Host "✅ .env file already exists" -ForegroundColor Green
}

# Build and start the service
Write-Host ""
Write-Host "Starting IB Bridge service..." -ForegroundColor Yellow
Write-Host ""

docker compose -f root-files/docker-compose.yml up -d ib-bridge

if ($LASTEXITCODE -eq 0) {
    Write-Host ""
    Write-Host "✅ IB Bridge service started successfully!" -ForegroundColor Green
    Write-Host ""
    
    # Wait a moment for service to start
    Write-Host "Waiting for service to initialize..." -ForegroundColor Yellow
    Start-Sleep -Seconds 5
    
    # Test the connection
    Write-Host ""
    Write-Host "Testing connection..." -ForegroundColor Yellow
    try {
        $response = Invoke-RestMethod -Uri "http://localhost:8092/health" -Method Get -TimeoutSec 5
        Write-Host ""
        Write-Host "✅ Service is healthy!" -ForegroundColor Green
        Write-Host "   Status: $($response.status)" -ForegroundColor Cyan
        Write-Host "   Connected to IB: $($response.connected)" -ForegroundColor Cyan
        Write-Host "   Version: $($response.version)" -ForegroundColor Cyan
    } catch {
        Write-Host ""
        Write-Host "⚠️  Service started but health check failed" -ForegroundColor Yellow
        Write-Host "   This might be normal if IB Gateway is not connected yet" -ForegroundColor Yellow
    }
    
    Write-Host ""
    Write-Host "========================================" -ForegroundColor Cyan
    Write-Host "  Next Steps" -ForegroundColor Cyan
    Write-Host "========================================" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "1. View logs:" -ForegroundColor White
    Write-Host "   docker compose -f root-files/docker-compose.yml logs -f ib-bridge" -ForegroundColor Gray
    Write-Host ""
    Write-Host "2. Test the API:" -ForegroundColor White
    Write-Host "   curl http://localhost:8092/health" -ForegroundColor Gray
    Write-Host "   curl http://localhost:8092/quotes/AAPL" -ForegroundColor Gray
    Write-Host ""
    Write-Host "3. Run Python tests:" -ForegroundColor White
    Write-Host "   cd services\ib-bridge" -ForegroundColor Gray
    Write-Host "   python test_bridge.py" -ForegroundColor Gray
    Write-Host ""
    Write-Host "4. Stop the service:" -ForegroundColor White
    Write-Host "   docker compose -f root-files/docker-compose.yml down ib-bridge" -ForegroundColor Gray
    Write-Host ""
    Write-Host "📚 Full documentation: services\ib-bridge\TESTING.md" -ForegroundColor Cyan
    Write-Host ""
    
} else {
    Write-Host ""
    Write-Host "❌ Failed to start IB Bridge service" -ForegroundColor Red
    Write-Host "   Check the logs: docker compose -f root-files/docker-compose.yml logs ib-bridge" -ForegroundColor Yellow
    exit 1
}
