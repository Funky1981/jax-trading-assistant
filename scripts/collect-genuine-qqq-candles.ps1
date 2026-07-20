<#
.SYNOPSIS
Safely retrieves and persists genuine historical QQQ candles.

.DESCRIPTION
Uses Jax's configured read-only market-data providers. It never submits an
order or changes approval, execution-worker, or live-trading settings.
#>
[CmdletBinding()]
param(
    [datetimeoffset]$From = [datetimeoffset]::Parse("2026-07-17T16:46:31Z"),
    [ValidateSet("1h", "1d")][string]$Timeframe = "1h",
    [ValidateRange(1, 1000)][int]$Limit = 168,
    [string]$ApiUrl = "http://localhost:8081",
    [string]$Operator = $env:USERNAME
)
$ErrorActionPreference="Stop";$ApiUrl=$ApiUrl.TrimEnd('/');$repo=Resolve-Path(Join-Path $PSScriptRoot "..");$headers=@{"X-User-ID"=$Operator}
function Get-ConfigValue([string]$Name){$v=[Environment]::GetEnvironmentVariable($Name);if(-not[string]::IsNullOrWhiteSpace($v)){return $v.Trim()};$p=Join-Path $repo ".env";if(-not(Test-Path -LiteralPath $p)){return $null};$line=Get-Content -LiteralPath $p|Where-Object{$_ -match "^$([regex]::Escape($Name))="}|Select-Object -First 1;if($null-eq$line){return $null};return(($line-split"=",2)[1]).Trim().Trim('"')}
try{$auth=Invoke-RestMethod -Method Get -Uri "$ApiUrl/auth/status" -TimeoutSec 10}catch{throw "Jax is unavailable at $ApiUrl. $($_.Exception.Message)"}
if($auth.enabled){$u=Get-ConfigValue "AUTH_BOOTSTRAP_USERNAME";$p=Get-ConfigValue "AUTH_BOOTSTRAP_PASSWORD";if([string]::IsNullOrWhiteSpace($u)-or[string]::IsNullOrWhiteSpace($p)){throw "Local operator credentials are not configured."};$login=Invoke-RestMethod -Method Post -Uri "$ApiUrl/auth/login" -ContentType "application/json" -Body(@{username=$u;password=$p}|ConvertTo-Json);$headers.Authorization="Bearer $($login.access_token)"}
$runtime=Invoke-RestMethod -Method Get -Uri "$ApiUrl/api/v1/system/runtime" -Headers $headers -TimeoutSec 10
if($runtime.runtimeMode -ne "paper" -or $runtime.allowLiveTrading -eq $true -or $runtime.executionInstructionWorkerEnabled -eq $true){throw "Safety check failed: Jax is not in paper/read-only mode or the execution worker is enabled."}
$body=@{symbol="QQQ";timeframe=$Timeframe;from=$From.UtcDateTime.ToString("o");limit=$Limit}|ConvertTo-Json
Write-Host "QQQ genuine candle collection" -ForegroundColor Cyan;Write-Host "  Mode: paper/read-only";Write-Host "  Requested: historical $Timeframe from $($From.UtcDateTime.ToString('u'))"
try{$result=Invoke-RestMethod -Method Post -Uri "$ApiUrl/api/v1/market/candles/collect" -Headers $headers -ContentType "application/json" -Body $body -TimeoutSec 60}catch{$d=if($_.ErrorDetails.Message){$_.ErrorDetails.Message.Trim()}else{$_.Exception.Message};throw "Genuine QQQ collection failed: $d"}
if($result.provider -in @("TEST","SYNTHETIC","FIXTURE","unknown",$null)){throw "Safety check failed: non-genuine provider provenance returned."}
Write-Host "  Provider: $($result.provider)" -ForegroundColor Green;Write-Host "  Provider classification: $($result.marketDataClassification)";Write-Host "  Timeframe: $($result.timeframe) ($($result.timestampSemantics))";$rth=if($null-eq$result.regularTradingHours){"unknown"}else{[string]$result.regularTradingHours};Write-Host "  Regular-hours only: $rth";Write-Host "  Received / persisted: $($result.received) / $($result.persisted)";Write-Host "  Earliest / latest: $($result.earliest) / $($result.latest)"
if($result.received -eq 0){Write-Host "No genuine observations covered the requested period; no intervals were fabricated." -ForegroundColor Yellow}
