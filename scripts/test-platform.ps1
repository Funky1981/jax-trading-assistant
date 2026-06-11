param(
  [ValidateSet("quick", "full")]
  [string]$Mode = "quick",
  [string]$ApiBase = "http://localhost:8081",
  [string]$ResearchBase = "http://localhost:8091",
  [string]$IbBridgeBase = "http://localhost:8092",
  [string]$Agent0Base = "http://localhost:8093",
  [string]$OutputDir = "Docs/runs",
  [switch]$OpenVisualReport
)

$ErrorActionPreference = "Stop"

$results = @()
$script:apiHeaders = @{}

function Get-ConfigValue {
  param([string]$Name)

  $value = [Environment]::GetEnvironmentVariable($Name)
  if (-not [string]::IsNullOrWhiteSpace($value)) {
    return $value.Trim()
  }

  $envPath = Join-Path (Get-Location) ".env"
  if (-not (Test-Path $envPath)) {
    return $null
  }

  $line = Get-Content $envPath | Where-Object { $_ -match "^$([regex]::Escape($Name))=" } | Select-Object -First 1
  if ($null -eq $line) {
    return $null
  }

  return (($line -split '=', 2)[1]).Trim().Trim('"')
}

function Add-Result {
  param(
    [string]$Step,
    [string]$Status,
    [string]$Detail
  )

  $script:results += [pscustomobject]@{
    Step   = $Step
    Status = $Status
    Detail = $Detail
  }

  $color = "Gray"
  if ($Status -eq "PASS") { $color = "Green" }
  if ($Status -eq "WARN") { $color = "Yellow" }
  if ($Status -eq "FAIL") { $color = "Red" }
  Write-Host ("[{0}] {1} - {2}" -f $Status, $Step, $Detail) -ForegroundColor $color
}

function Initialize-ApiAuth {
  param([string]$BaseUrl)

  $script:apiHeaders = @{}

  try {
    $authStatus = Invoke-RestMethod -Method GET -Uri "$BaseUrl/auth/status" -TimeoutSec 10
  } catch {
    Add-Result -Step "auth/status" -Status "WARN" -Detail "unable to determine auth mode: $($_.Exception.Message)"
    return
  }

  if (-not $authStatus.enabled) {
    Add-Result -Step "auth/status" -Status "PASS" -Detail "auth disabled"
    return
  }

  $username = Get-ConfigValue -Name "AUTH_BOOTSTRAP_USERNAME"
  $password = Get-ConfigValue -Name "AUTH_BOOTSTRAP_PASSWORD"
  if ([string]::IsNullOrWhiteSpace($username) -or [string]::IsNullOrWhiteSpace($password)) {
    Add-Result -Step "auth/login" -Status "FAIL" -Detail "AUTH_BOOTSTRAP credentials are not configured"
    return
  }

  try {
    $body = @{ username = $username; password = $password } | ConvertTo-Json
    $login = Invoke-RestMethod -Method POST -Uri "$BaseUrl/auth/login" -ContentType "application/json" -Body $body -TimeoutSec 10
    if ([string]::IsNullOrWhiteSpace($login.access_token)) {
      throw "access_token missing from login response"
    }
    $script:apiHeaders = @{ Authorization = "Bearer $($login.access_token)" }
    Add-Result -Step "auth/login" -Status "PASS" -Detail "auth enabled; bootstrap login succeeded"
  } catch {
    Add-Result -Step "auth/login" -Status "FAIL" -Detail $_.Exception.Message
  }
}

function Invoke-HttpCheck {
  param(
    [string]$Name,
    [string]$Url
  )

  try {
    $null = Invoke-RestMethod -Method GET -Uri $Url -Headers $script:apiHeaders -TimeoutSec 15
    Add-Result -Step $Name -Status "PASS" -Detail $Url
  } catch {
    Add-Result -Step $Name -Status "FAIL" -Detail $_.Exception.Message
  }
}

function Invoke-CommandStep {
  param(
    [string]$Step,
    [scriptblock]$Action
  )

  try {
    & $Action
    if ($LASTEXITCODE -ne 0) {
      throw "command exited with code $LASTEXITCODE"
    }
    Add-Result -Step $Step -Status "PASS" -Detail "completed"
  } catch {
    Add-Result -Step $Step -Status "FAIL" -Detail $_.Exception.Message
  }
}

function Ensure-Tool {
  param([string]$Name)
  if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
    throw "$Name is not available on PATH."
  }
}

function Write-RunReport {
  param([string]$Folder)

  if (-not (Test-Path $Folder)) {
    New-Item -Path $Folder -ItemType Directory | Out-Null
  }

  $stamp = Get-Date -Format "yyyyMMdd_HHmmss"
  $jsonPath = Join-Path $Folder "test_run_$stamp.json"
  $mdPath = Join-Path $Folder "test_run_$stamp.md"

  $results | ConvertTo-Json -Depth 6 | Set-Content -Path $jsonPath

  $lines = @()
  $lines += "# Platform Test Run"
  $lines += ""
  $lines += "- Mode: $Mode"
  $lines += "- Timestamp: $(Get-Date -Format s)"
  $lines += ""
  $lines += "## Results"
  foreach ($r in $results) {
    $lines += ("- [{0}] {1}: {2}" -f $r.Status, $r.Step, $r.Detail)
  }
  $lines += ""
  $lines += "## Endpoints"
  $lines += "- Trader API: $ApiBase"
  $lines += "- Research: $ResearchBase"
  $lines += "- IB Bridge: $IbBridgeBase"
  $lines += "- Agent0 Service: $Agent0Base"
  $lines | Set-Content -Path $mdPath

  Write-Host ""
  Write-Host "Report written:" -ForegroundColor Cyan
  Write-Host "  $mdPath"
  Write-Host "  $jsonPath"
}

Ensure-Tool -Name "go"

Write-Host "== platform tests ($Mode) ==" -ForegroundColor Cyan
Initialize-ApiAuth -BaseUrl $ApiBase

# 1) Service health checks
Invoke-HttpCheck -Name "health/trader-api" -Url "$ApiBase/health"
Invoke-HttpCheck -Name "health/research" -Url "$ResearchBase/health"
Invoke-HttpCheck -Name "health/ib-bridge" -Url "$IbBridgeBase/health"
Invoke-HttpCheck -Name "health/agent0-service" -Url "$Agent0Base/health"

# 2) API smoke checks (read-only)
Invoke-HttpCheck -Name "api/signals" -Url "$ApiBase/api/v1/signals?limit=1"
Invoke-HttpCheck -Name "api/artifacts" -Url "$ApiBase/api/v1/artifacts"
Invoke-HttpCheck -Name "api/testing-status" -Url "$ApiBase/api/v1/testing/status"
Invoke-HttpCheck -Name "api/testing-readiness" -Url "$ApiBase/api/v1/testing/readiness"
Invoke-HttpCheck -Name "api/runs" -Url "$ApiBase/api/v1/runs?limit=5"
Invoke-HttpCheck -Name "api/ai-decisions" -Url "$ApiBase/api/v1/ai-decisions?limit=5"
Invoke-HttpCheck -Name "api/etf-instruments" -Url "$ApiBase/api/v1/instruments/etfs"
Invoke-HttpCheck -Name "api/trading-pilot-status" -Url "$ApiBase/api/v1/trading/pilot-status"
Invoke-HttpCheck -Name "api/robust-performance" -Url "$ApiBase/api/v1/robust/performance"

# 3) Backend checks
if ($Mode -eq "quick") {
  Invoke-CommandStep -Step "go-verify quick (critical packages)" -Action {
    & "scripts/go-verify.ps1" -Mode quick -Packages "./cmd/trader" "./cmd/research" "./internal/strategyregistry" "./tests/golden"
  }
  Invoke-CommandStep -Step "golden utility tests" -Action {
    go test ./tests/golden -count=1
  }
} else {
  Invoke-CommandStep -Step "go-verify full" -Action {
    & "scripts/go-verify.ps1" -Mode full
  }
  Invoke-CommandStep -Step "golden verify" -Action {
    & "scripts/golden-check.ps1" -Mode verify
  }
}

Invoke-CommandStep -Step "ETF backend validation" -Action {
  go test ./internal/modules/instruments/... ./internal/modules/candidates/... ./internal/modules/approvals/... ./internal/modules/execution/... ./cmd/trader/...
}

# 4) Frontend checks
if (Test-Path "frontend/package.json") {
  Invoke-CommandStep -Step "frontend lint/type/test" -Action {
    Push-Location "frontend"
    try {
      npm run lint
      if ($LASTEXITCODE -ne 0) { throw "npm run lint failed with code $LASTEXITCODE" }
      npm run typecheck
      if ($LASTEXITCODE -ne 0) { throw "npm run typecheck failed with code $LASTEXITCODE" }
      npm run test
      if ($LASTEXITCODE -ne 0) { throw "npm run test failed with code $LASTEXITCODE" }
    } finally {
      Pop-Location
    }
  }

  if ($Mode -eq "full") {
    Invoke-CommandStep -Step "frontend e2e (playwright html report)" -Action {
      Push-Location "frontend"
      try {
        npx playwright test --reporter=html
      } finally {
        Pop-Location
      }
    }
    if ($OpenVisualReport.IsPresent) {
      Invoke-CommandStep -Step "open playwright html report" -Action {
        Push-Location "frontend"
        try {
          npx playwright show-report
        } finally {
          Pop-Location
        }
      }
    }
  }
} else {
  Add-Result -Step "frontend checks" -Status "WARN" -Detail "frontend/package.json not found"
}

Write-RunReport -Folder $OutputDir
