<#
.SYNOPSIS
Tracks hypothetical research outcomes for an approved paper ticket.

.DESCRIPTION
Calculates due 1-hour, 1-day, and 1-week checkpoints exclusively from market
data already persisted by Jax. It does not create a fill, instruction, order,
or trade. Missing data is reported as pending rather than estimated.

.EXAMPLE
.\scripts\track-paper-ticket-outcomes.ps1 `
  -PaperTicketId "pt_39cb350a-d796-4f7e-8c05-d746acb6b1d6"
#>
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true, Position = 0)]
    [ValidatePattern('^pt_[0-9a-fA-F-]{36}$')]
    [string]$PaperTicketId,
    [ValidateNotNullOrEmpty()]
    [string]$ApiUrl = "http://localhost:8081",
    [ValidateNotNullOrEmpty()]
    [string]$Operator = $env:USERNAME
)

$ErrorActionPreference = "Stop"
$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$ApiUrl = $ApiUrl.TrimEnd('/')
$headers = @{ "X-User-ID" = $Operator }

function Get-ConfigValue([string]$Name) {
    $value = [Environment]::GetEnvironmentVariable($Name)
    if (-not [string]::IsNullOrWhiteSpace($value)) { return $value.Trim() }
    $envPath = Join-Path $RepoRoot ".env"
    if (-not (Test-Path -LiteralPath $envPath)) { return $null }
    $line = Get-Content -LiteralPath $envPath | Where-Object { $_ -match "^$([regex]::Escape($Name))=" } | Select-Object -First 1
    if ($null -eq $line) { return $null }
    return (($line -split "=", 2)[1]).Trim().Trim('"')
}

try { $auth = Invoke-RestMethod -Method Get -Uri "$ApiUrl/auth/status" -TimeoutSec 10 }
catch { throw "Jax is not reachable at $ApiUrl. Start the local stack after applying migrations, then try again. $($_.Exception.Message)" }
if ($auth.enabled) {
    $username = Get-ConfigValue "AUTH_BOOTSTRAP_USERNAME"
    $password = Get-ConfigValue "AUTH_BOOTSTRAP_PASSWORD"
    if ([string]::IsNullOrWhiteSpace($username) -or [string]::IsNullOrWhiteSpace($password)) { throw "Local operator credentials are not configured." }
    $login = Invoke-RestMethod -Method Post -Uri "$ApiUrl/auth/login" -ContentType "application/json" -Body (@{ username=$username; password=$password } | ConvertTo-Json) -TimeoutSec 10
    $headers.Authorization = "Bearer $($login.access_token)"
}

try { $result = Invoke-RestMethod -Method Post -Uri "$ApiUrl/api/v1/paper-tickets/$PaperTicketId/outcomes/track" -Headers $headers -TimeoutSec 30 }
catch { $detail = if ($_.ErrorDetails.Message) { $_.ErrorDetails.Message.Trim() } else { $_.Exception.Message }; throw "Jax could not track this paper ticket: $detail" }

if ($result.nature -ne "hypothetical_research_outcome") { throw "Safety verification failed: response was not labelled hypothetical." }
if ($result.executionInstructions -ne 0 -or $result.orderIntents -ne 0 -or $result.brokerOrders -ne 0 -or $result.trades -ne 0) { throw "SAFETY VERIFICATION FAILED: execution-linked state exists. Stop and investigate." }

Write-Host ""
Write-Host "Hypothetical paper-ticket outcome" -ForegroundColor Cyan
Write-Host "  Ticket: $($result.paperTicketId)"
Write-Host "  Nature: hypothetical research outcome (not a broker trade)" -ForegroundColor Yellow
Write-Host "  Entry: $($result.entryAssumption)"
foreach ($checkpoint in $result.checkpoints) {
    Write-Host ""
    $statusColour = if ($checkpoint.checkpointStatus -like 'pending*') { 'Yellow' } elseif ($checkpoint.checkpointStatus -in @('invalid_ticket', 'cancelled', 'insufficient_data')) { 'Red' } else { 'Green' }
    Write-Host "  Checkpoint $($checkpoint.checkpointName): $($checkpoint.checkpointStatus)" -ForegroundColor $statusColour
    Write-Host "    Scheduled: $($checkpoint.scheduledAt)"
    Write-Host "    Tracking start: $($checkpoint.trackingStartedAt) ($($checkpoint.trackingStartSource))"
    Write-Host "    Observation: $(if ($checkpoint.observationAt) { $checkpoint.observationAt } else { 'not available' })"
    Write-Host "    Observation delay (seconds): $(if ($null -ne $checkpoint.observationDelaySeconds) { $checkpoint.observationDelaySeconds } else { 'pending' })"
    Write-Host "    Price / hypothetical P&L: $(if ($null -ne $checkpoint.checkpointPrice) { "$($checkpoint.checkpointPrice) / $($checkpoint.hypotheticalPnl)" } else { 'pending' })"
    Write-Host "    Price change / return: $(if ($null -ne $checkpoint.priceChange) { "$($checkpoint.priceChange) / $($checkpoint.percentageReturn)%" } else { 'pending' })"
    Write-Host "    Range / MFE / MAE: $($checkpoint.lowestObservedPrice) - $($checkpoint.highestObservedPrice) / $($checkpoint.maximumFavourableExcursion) / $($checkpoint.maximumAdverseExcursion)"
    Write-Host "    Stop / target touched: $($checkpoint.stopTouched) / $($checkpoint.targetTouched)"
    Write-Host "    First stop / target touch: $($checkpoint.firstStopTouchAt) / $($checkpoint.firstTargetTouchAt)"
    Write-Host "    Observation range: $($checkpoint.earliestObservationAt) - $($checkpoint.latestObservationAt)"
    Write-Host "    Data: $($checkpoint.marketDataSource), $($checkpoint.marketDataClassification), $($checkpoint.candleInterval), observations=$($checkpoint.observationCount), quality=$($checkpoint.dataQualityStatus)"
}
Write-Host ""
Write-Host "Safety verified: no execution instruction, order intent, broker order, or trade was created." -ForegroundColor Green
