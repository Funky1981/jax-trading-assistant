param(
  [string]$ApiBase = "http://localhost:8081",
  [string]$OutputDir = "Docs/runs/etf-paper-pilot",
  [string]$PilotSymbol = "SPY",
  [string]$ExcludedSymbol = "TQQQ",
  [switch]$AutomatedValidationPassed,
  [switch]$OperatorUATPassed,
  [switch]$PaperPilotSignedOff,
  [switch]$EngineeringSignoff,
  [switch]$OperationsSignoff,
  [switch]$TradingRiskSignoff
)

$ErrorActionPreference = "Stop"

$results = @()
$script:apiHeaders = @{}

function Get-ConfigValue {
  param(
    [string]$Name
  )

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

function Initialize-ApiAuth {
  param(
    [string]$BaseUrl
  )

  $script:apiHeaders = @{}

  try {
    $authStatus = Invoke-RestMethod -Method GET -Uri "$BaseUrl/auth/status" -TimeoutSec 10
  } catch {
    Add-Check -Name "auth/status" -Status "WARN" -Detail "unable to determine auth mode: $($_.Exception.Message)"
    return
  }

  if (-not $authStatus.enabled) {
    Add-Check -Name "auth/status" -Status "PASS" -Detail "auth disabled"
    return
  }

  $username = Get-ConfigValue -Name "AUTH_BOOTSTRAP_USERNAME"
  $password = Get-ConfigValue -Name "AUTH_BOOTSTRAP_PASSWORD"
  if ([string]::IsNullOrWhiteSpace($username) -or [string]::IsNullOrWhiteSpace($password)) {
    Add-Check -Name "auth/login" -Status "FAIL" -Detail "AUTH_BOOTSTRAP credentials are not configured"
    return
  }

  try {
    $body = @{ username = $username; password = $password } | ConvertTo-Json
    $login = Invoke-RestMethod -Method POST -Uri "$BaseUrl/auth/login" -ContentType "application/json" -Body $body -TimeoutSec 10
    if ([string]::IsNullOrWhiteSpace($login.access_token)) {
      throw "access_token missing from login response"
    }
    $script:apiHeaders = @{ Authorization = "Bearer $($login.access_token)" }
    Add-Check -Name "auth/login" -Status "PASS" -Detail "auth enabled; bootstrap login succeeded"
  } catch {
    Add-Check -Name "auth/login" -Status "FAIL" -Detail $_.Exception.Message
  }
}

function Add-Check {
  param(
    [string]$Name,
    [string]$Status,
    [string]$Detail
  )

  $script:results += [pscustomobject]@{
    name   = $Name
    status = $Status
    detail = $Detail
  }

  $color = "Gray"
  if ($Status -eq "PASS") { $color = "Green" }
  if ($Status -eq "WARN") { $color = "Yellow" }
  if ($Status -eq "FAIL") { $color = "Red" }
  Write-Host ("[{0}] {1} - {2}" -f $Status, $Name, $Detail) -ForegroundColor $color
}

function Get-Json {
  param(
    [string]$Name,
    [string]$Url
  )

  try {
    $response = Invoke-RestMethod -Method GET -Uri $Url -Headers $script:apiHeaders -TimeoutSec 20
    Add-Check -Name $Name -Status "PASS" -Detail $Url
    return $response
  } catch {
    Add-Check -Name $Name -Status "FAIL" -Detail $_.Exception.Message
    return $null
  }
}

function Test-Switch {
  param(
    [string]$Name,
    [bool]$Value
  )

  if ($Value) {
    Add-Check -Name $Name -Status "PASS" -Detail "provided"
  } else {
    Add-Check -Name $Name -Status "WARN" -Detail "not provided"
  }
}

if (-not (Test-Path $OutputDir)) {
  New-Item -Path $OutputDir -ItemType Directory | Out-Null
}

$stamp = Get-Date -Format "yyyyMMdd_HHmmss"
Initialize-ApiAuth -BaseUrl $ApiBase
$catalog = Get-Json -Name "etf/catalog" -Url "$ApiBase/api/v1/instruments/etfs"
$pilotStatus = Get-Json -Name "etf/pilot-status" -Url "$ApiBase/api/v1/trading/pilot-status"
$readiness = Get-Json -Name "etf/testing-readiness" -Url "$ApiBase/api/v1/testing/readiness"

if ($catalog -ne $null) {
  if ($catalog.version -eq "phase1-2026-05-13") {
    Add-Check -Name "catalog/version" -Status "PASS" -Detail $catalog.version
  } else {
    Add-Check -Name "catalog/version" -Status "FAIL" -Detail "expected phase1-2026-05-13, got $($catalog.version)"
  }

  $pilotInstrument = @($catalog.instruments | Where-Object { $_.symbol -eq $PilotSymbol })
  if ($pilotInstrument.Count -gt 0) {
    Add-Check -Name "catalog/pilot-symbol" -Status "PASS" -Detail "$PilotSymbol present"
  } else {
    Add-Check -Name "catalog/pilot-symbol" -Status "FAIL" -Detail "$PilotSymbol missing"
  }

  $excludedInstrument = @($catalog.instruments | Where-Object { $_.symbol -eq $ExcludedSymbol })
  if ($excludedInstrument.Count -gt 0 -and @($excludedInstrument[0].exclusions).Count -gt 0) {
    Add-Check -Name "catalog/excluded-symbol" -Status "PASS" -Detail "$ExcludedSymbol has exclusions"
  } else {
    Add-Check -Name "catalog/excluded-symbol" -Status "FAIL" -Detail "$ExcludedSymbol exclusion evidence missing"
  }
}

if ($pilotStatus -ne $null) {
  if ($pilotStatus.etfPhase1Enabled -eq $true) {
    Add-Check -Name "pilot/etf-phase1" -Status "PASS" -Detail "enabled"
  } else {
    Add-Check -Name "pilot/etf-phase1" -Status "FAIL" -Detail "etfPhase1Enabled is not true"
  }

  if ($pilotStatus.etfEntryWorkflow -eq "candidate_approval_only") {
    Add-Check -Name "pilot/workflow" -Status "PASS" -Detail $pilotStatus.etfEntryWorkflow
  } else {
    Add-Check -Name "pilot/workflow" -Status "FAIL" -Detail "expected candidate_approval_only, got $($pilotStatus.etfEntryWorkflow)"
  }
}

if ($readiness -ne $null -and $readiness.etfPhase1Readiness -ne $null) {
  $etfReadiness = $readiness.etfPhase1Readiness
  if ($etfReadiness.catalogLoaded -eq $true) {
    Add-Check -Name "readiness/catalog-loaded" -Status "PASS" -Detail "catalog loaded"
  } else {
    Add-Check -Name "readiness/catalog-loaded" -Status "FAIL" -Detail "catalog not loaded"
  }

  Add-Check -Name "readiness/status" -Status "PASS" -Detail $etfReadiness.status
} else {
  Add-Check -Name "readiness/etf-section" -Status "FAIL" -Detail "etfPhase1Readiness missing"
}

Test-Switch -Name "signoff/automated-validation" -Value $AutomatedValidationPassed.IsPresent
Test-Switch -Name "signoff/operator-uat" -Value $OperatorUATPassed.IsPresent
Test-Switch -Name "signoff/paper-pilot" -Value $PaperPilotSignedOff.IsPresent
Test-Switch -Name "signoff/engineering" -Value $EngineeringSignoff.IsPresent
Test-Switch -Name "signoff/operations" -Value $OperationsSignoff.IsPresent
Test-Switch -Name "signoff/trading-risk" -Value $TradingRiskSignoff.IsPresent

$failed = @($results | Where-Object { $_.status -eq "FAIL" }).Count
$warnings = @($results | Where-Object { $_.status -eq "WARN" }).Count
$status = "ready_for_review"
if ($failed -gt 0) {
  $status = "blocked"
} elseif ($warnings -gt 0) {
  $status = "awaiting_signoff"
}

$report = [pscustomobject]@{
  status      = $status
  generatedAt = (Get-Date).ToUniversalTime().ToString("o")
  apiBase     = $ApiBase
  pilotSymbol = $PilotSymbol
  excludedSymbol = $ExcludedSymbol
  checks      = $results
  catalog     = $catalog
  pilotStatus = $pilotStatus
  readiness   = $readiness
}

$jsonPath = Join-Path $OutputDir "etf_pilot_evidence_$stamp.json"
$mdPath = Join-Path $OutputDir "etf_pilot_evidence_$stamp.md"
$report | ConvertTo-Json -Depth 12 | Set-Content -Path $jsonPath

$lines = @()
$lines += "# ETF Paper Pilot Evidence"
$lines += ""
$lines += "- status: $status"
$lines += "- generated: $($report.generatedAt)"
$lines += "- api_base: $ApiBase"
$lines += "- pilot_symbol: $PilotSymbol"
$lines += "- excluded_symbol: $ExcludedSymbol"
$lines += ""
$lines += "## Checks"
foreach ($check in $results) {
  $lines += ("- [{0}] {1}: {2}" -f $check.status, $check.name, $check.detail)
}
$lines += ""
$lines += "## Sign-Off Environment"
$lines += ""
$lines += "Set these only after this report and related UAT evidence are reviewed:"
$lines += ""
$lines += '```powershell'
$lines += '$env:ETF_PHASE1_AUTOMATED_VALIDATION="passed"'
$lines += '$env:ETF_PHASE1_OPERATOR_UAT="passed"'
$lines += '$env:ETF_PHASE1_PAPER_PILOT_SIGNOFF="passed"'
$lines += '$env:ETF_PHASE1_ENGINEERING_SIGNOFF="true"'
$lines += '$env:ETF_PHASE1_OPERATIONS_SIGNOFF="true"'
$lines += '$env:ETF_PHASE1_TRADING_RISK_SIGNOFF="true"'
$lines += '```'
$lines | Set-Content -Path $mdPath

Write-Host ""
Write-Host "ETF pilot evidence written:" -ForegroundColor Cyan
Write-Host "  $mdPath"
Write-Host "  $jsonPath"

if ($failed -gt 0) {
  exit 1
}
