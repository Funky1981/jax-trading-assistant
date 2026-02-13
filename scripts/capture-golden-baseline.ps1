# Capture Golden Baseline
# This script captures the current behavior of the system as a golden baseline

Write-Host "🎯 Capturing Golden Baseline..." -ForegroundColor Cyan

# Check if services are running
Write-Host "`nChecking if services are running..." -ForegroundColor Yellow

$services = @(
    @{Name="jax-signal-generator"; Port=8096; Required=$true},
    @{Name="jax-api"; Port=8081; Required=$true},
    @{Name="postgres"; Port=5433; Required=$true}
)

$allRunning = $true
foreach ($service in $services) {
    try {
        $connection = Test-NetConnection -ComputerName localhost -Port $service.Port -WarningAction SilentlyContinue -InformationLevel Quiet
        if ($connection) {
            Write-Host "  ✅ $($service.Name) (port $($service.Port))" -ForegroundColor Green
        } else {
            if ($service.Required) {
                Write-Host "  ❌ $($service.Name) (port $($service.Port)) - NOT RUNNING" -ForegroundColor Red
                $allRunning = $false
            } else {
                Write-Host "  ⚠️  $($service.Name) (port $($service.Port)) - NOT RUNNING (optional)" -ForegroundColor Yellow
            }
        }
    } catch {
        if ($service.Required) {
            Write-Host "  ❌ $($service.Name) - CANNOT CHECK" -ForegroundColor Red
            $allRunning = $false
        }
    }
}

if (-not $allRunning) {
    Write-Host "`n❌ Not all required services are running." -ForegroundColor Red
    Write-Host "Please start services first:" -ForegroundColor Yellow
    Write-Host "  docker-compose up -d" -ForegroundColor Cyan
    exit 1
}

Write-Host "`n✅ All required services are running`n" -ForegroundColor Green

# Wait a bit for services to stabilize
Write-Host "Waiting for services to stabilize..." -ForegroundColor Yellow
Start-Sleep -Seconds 5

# Run the capture tool
Write-Host "`nRunning capture tool..." -ForegroundColor Cyan
try {
    $output = go run .\tests\golden\cmd\capture.go 2>&1
    Write-Host $output
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "`n🎉 Golden baseline captured successfully!" -ForegroundColor Green
        
        # List captured files
        Write-Host "`nCaptured files:" -ForegroundColor Cyan
        Get-ChildItem -Path tests\golden -Recurse -Filter "baseline-*.json" | ForEach-Object {
            $size = "{0:N2}" -f ($_.Length / 1KB)
            Write-Host "  📄 $($_.FullName.Replace($PWD, '.')) ($size KB)" -ForegroundColor Gray
        }
        
        Write-Host "`nNext steps:" -ForegroundColor Yellow
        Write-Host "  1. Review the captured files to ensure they look correct"
        Write-Host "  2. Commit these files to version control"
        Write-Host "  3. Run golden tests: go test -v ./tests/golden/... -tags=golden"
        
    } else {
        Write-Host "`n❌ Capture failed with exit code $LASTEXITCODE" -ForegroundColor Red
        exit $LASTEXITCODE
    }
} catch {
    Write-Host "`n❌ Error running capture: $_" -ForegroundColor Red
    exit 1
}
