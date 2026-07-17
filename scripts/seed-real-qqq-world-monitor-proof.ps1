param(
  [string]$ApiBase = "http://localhost:8081"
)

$ErrorActionPreference = "Stop"
$headers = @{}

function Get-ConfigValue([string]$Name) {
  $value = [Environment]::GetEnvironmentVariable($Name)
  if (-not [string]::IsNullOrWhiteSpace($value)) { return $value.Trim() }
  if (-not (Test-Path ".env")) { return $null }
  $line = Get-Content ".env" | Where-Object { $_ -match "^$([regex]::Escape($Name))=" } | Select-Object -First 1
  if ($null -eq $line) { return $null }
  return (($line -split "=", 2)[1]).Trim().Trim('"')
}

$authStatus = Invoke-RestMethod -Method GET -Uri "$ApiBase/auth/status" -TimeoutSec 10
if ($authStatus.enabled) {
  $username = Get-ConfigValue "AUTH_BOOTSTRAP_USERNAME"
  $password = Get-ConfigValue "AUTH_BOOTSTRAP_PASSWORD"
  if ([string]::IsNullOrWhiteSpace($username) -or [string]::IsNullOrWhiteSpace($password)) {
    throw "AUTH_BOOTSTRAP_USERNAME and AUTH_BOOTSTRAP_PASSWORD are required"
  }
  $login = Invoke-RestMethod -Method POST -Uri "$ApiBase/auth/login" -ContentType "application/json" -Body (@{
    username = $username
    password = $password
  } | ConvertTo-Json) -TimeoutSec 10
  $headers = @{ Authorization = "Bearer $($login.access_token)" }
}

$sourceEventId = "real-qqq-proof-$([Guid]::NewGuid().ToString('N'))"
$trigger = @{
  source = "world-monitor-local-proof"
  is_synthetic = $true
  source_event_id = $sourceEventId
  event_type = "macro_rates"
  headline = "Local proof event: softer inflation supports QQQ research review"
  summary = "Operator-generated local proof input for the real World Monitor normalization and paper-candidate pipeline."
  source_urls = @(
    "https://www.bls.gov/cpi/",
    "https://www.federalreserve.gov/monetarypolicy.htm"
  )
  source_count = 2
  timestamp_utc = (Get-Date).ToUniversalTime().AddMinutes(-1).ToString("o")
  region = "US"
  possible_affected_etfs = @("QQQ")
  asset_themes = @("inflation", "rates", "growth_equities")
  severity = "high"
  source_tier = "tier2"
  confidence = 0.74
  confidence_reasons = @(
    "two authoritative public sources",
    "macro rates event mapped to QQQ",
    "local proof input is fresh"
  )
  reason = "Rates-sensitive macro evidence can affect the QQQ research outlook."
  raw_payload = @{
    localProof = $true
    proofVersion = "real-qqq-world-monitor-v1"
  }
}

$receipt = Invoke-RestMethod -Method POST -Uri "$ApiBase/api/v1/research/events/world-monitor" -Headers $headers -ContentType "application/json" -Body ($trigger | ConvertTo-Json -Depth 8) -TimeoutSec 30
if ($receipt.status -ne "new" -or [string]::IsNullOrWhiteSpace($receipt.inbox_id) -or [string]::IsNullOrWhiteSpace($receipt.event_id)) {
  throw "World Monitor proof row was not normalized as new: $($receipt | ConvertTo-Json -Compress)"
}

[pscustomobject]@{
  sourceEventId = $sourceEventId
  inboxId = $receipt.inbox_id
  normalizedEventId = $receipt.event_id
  status = $receipt.status
  possibleAffectedETFs = @("QQQ")
  confidence = 0.74
  candidateId = $null
} | ConvertTo-Json -Depth 4
