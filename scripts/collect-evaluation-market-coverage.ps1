[CmdletBinding()]
param(
    [string]$ApiUrl = "http://127.0.0.1:8081",
    [string]$DecisionRuleset = "genuine-event-decision-v2",
    [string]$ResolverRuleset = "event-asset-resolution-v1",
    [int]$MaxSymbols = 25,
    [string]$AccessToken = "",
    [string]$Username = $env:AUTH_BOOTSTRAP_USERNAME,
    [string]$Password = $env:AUTH_BOOTSTRAP_PASSWORD
)
$ErrorActionPreference = "Stop"
if ($MaxSymbols -lt 1 -or $MaxSymbols -gt 25) { throw "MaxSymbols must be between 1 and 25" }
$headers = @{}
if ([string]::IsNullOrWhiteSpace($AccessToken) -and -not [string]::IsNullOrWhiteSpace($Username) -and -not [string]::IsNullOrWhiteSpace($Password)) {
    $loginBody = @{ username=$Username; password=$Password } | ConvertTo-Json
    $login = Invoke-RestMethod -Method Post -Uri "$ApiUrl/auth/login" -ContentType "application/json" -Body $loginBody -TimeoutSec 30
    $AccessToken = $login.access_token
}
if (-not [string]::IsNullOrWhiteSpace($AccessToken)) { $headers["Authorization"] = "Bearer $AccessToken" }
$body = @{ decisionRuleset=$DecisionRuleset; resolverRuleset=$ResolverRuleset; maxSymbols=$MaxSymbols } | ConvertTo-Json
Write-Host "Collecting bounded evaluation coverage in paper/read-only mode for at most $MaxSymbols accepted symbols"
Invoke-RestMethod -Method Post -Uri "$ApiUrl/api/v1/market/candles/collect-evaluation" -Headers $headers -ContentType "application/json" -Body $body -TimeoutSec 240
