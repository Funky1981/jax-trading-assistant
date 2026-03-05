param(
  [ValidateSet("quick", "full")]
  [string]$Mode = "quick",
  [string]$ApiBase = "http://localhost:8081",
  [string]$ResearchBase = "http://localhost:8091",
  [string]$IbBridgeBase = "http://localhost:8092",
  [string]$Agent0Base = "http://localhost:8093",
  [string]$HindsightBase = "http://localhost:8888",
  [string]$OutputDir = "Docs/runs",
  [switch]$OpenVisualReport
)

$ErrorActionPreference = "Stop"

$results = @()

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

function Invoke-HttpCheck {
  param(
    [string]$Name,
    [string]$Url
  )

  try {
    $null = Invoke-RestMethod -Method GET -Uri $Url -TimeoutSec 15
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
  $lines += "- Hindsight: $HindsightBase"
  $lines | Set-Content -Path $mdPath

  Write-Host ""
  Write-Host "Report written:" -ForegroundColor Cyan
  Write-Host "  $mdPath"
  Write-Host "  $jsonPath"
}

Ensure-Tool -Name "go"

Write-Host "== platform tests ($Mode) ==" -ForegroundColor Cyan

# 1) Service health checks
Invoke-HttpCheck -Name "health/trader-api" -Url "$ApiBase/health"
Invoke-HttpCheck -Name "health/research" -Url "$ResearchBase/health"
Invoke-HttpCheck -Name "health/ib-bridge" -Url "$IbBridgeBase/health"
Invoke-HttpCheck -Name "health/agent0-service" -Url "$Agent0Base/health"
Invoke-HttpCheck -Name "health/hindsight" -Url "$HindsightBase/metrics"

# 2) API smoke checks (read-only)
Invoke-HttpCheck -Name "api/signals" -Url "$ApiBase/api/v1/signals?limit=1"
Invoke-HttpCheck -Name "api/artifacts" -Url "$ApiBase/api/v1/artifacts"
Invoke-HttpCheck -Name "api/testing-status" -Url "$ApiBase/api/v1/testing/status"
Invoke-HttpCheck -Name "api/runs" -Url "$ApiBase/api/v1/runs?limit=5"
Invoke-HttpCheck -Name "api/ai-decisions" -Url "$ApiBase/api/v1/ai-decisions?limit=5"

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

# 4) Frontend checks
if (Test-Path "frontend/package.json") {
  Invoke-CommandStep -Step "frontend lint/type/test" -Action {
    Push-Location "frontend"
    try {
      npm run lint
      npm run typecheck
      npm run test
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
