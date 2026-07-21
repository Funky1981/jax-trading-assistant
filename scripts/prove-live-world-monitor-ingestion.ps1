[CmdletBinding()]
param(
    [switch]$ReviewOnly,
    [string]$JaxApiUrl = "http://localhost:8081"
)

$ErrorActionPreference = "Stop"
$jaxRoot = Split-Path -Parent $PSScriptRoot
$worldMonitorRoot = "C:\Projects\Jax-World-News-Monitor"
if (-not (Test-Path -LiteralPath $worldMonitorRoot -PathType Container)) { throw "World Monitor repository not found: $worldMonitorRoot" }
$token = $env:JAX_RESEARCH_TOKEN
if (-not $token) {
    $envLines = Get-Content -LiteralPath (Join-Path $jaxRoot ".env")
    $username = (($envLines | Where-Object { $_ -match '^AUTH_BOOTSTRAP_USERNAME=' }) -replace '^AUTH_BOOTSTRAP_USERNAME=', '').Trim()
    $password = (($envLines | Where-Object { $_ -match '^AUTH_BOOTSTRAP_PASSWORD=' }) -replace '^AUTH_BOOTSTRAP_PASSWORD=', '').Trim()
    if ($username -and $password) {
        $login = Invoke-RestMethod -Method Post -Uri "$($JaxApiUrl.TrimEnd('/'))/auth/login" -ContentType "application/json" -Body (@{ username = $username; password = $password } | ConvertTo-Json)
        $token = $login.access_token
    }
}

function Invoke-JaxGet([string]$Path) {
    $headers = @{}
    if ($token) { $headers.Authorization = "Bearer $token" }
    Invoke-RestMethod -Uri "$($JaxApiUrl.TrimEnd('/'))$Path" -Headers $headers -TimeoutSec 10
}
function Get-SafetyCounts {
    $sql = "SELECT json_build_object('automaticApprovals',COUNT(*) FILTER (WHERE approved_by <> 'human'),'executionInstructions',(SELECT COUNT(*) FROM execution_instructions),'orderIntents',(SELECT COUNT(*) FROM order_intents),'brokerOrders',(SELECT COUNT(*) FROM execution_instructions WHERE broker_order_id IS NOT NULL),'trades',(SELECT COUNT(*) FROM trades),'fills',(SELECT COUNT(*) FROM fills)) FROM candidate_approvals;"
    $json = docker compose exec -T postgres psql -U jax -d jax -Atc $sql
    if ($LASTEXITCODE -ne 0) { throw "Could not query Jax safety counts" }
    $json | ConvertFrom-Json
}

$health = Invoke-RestMethod -Uri "$($JaxApiUrl.TrimEnd('/'))/health" -TimeoutSec 10
$runtime = Invoke-JaxGet "/api/v1/system/runtime"
$maxLeverage = (Get-Content -LiteralPath (Join-Path $jaxRoot "config\risk-constraints.json") -Raw | ConvertFrom-Json).position_limits.max_leverage
$before = Get-SafetyCounts
if ($ReviewOnly) {
    $inbox = Invoke-JaxGet "/api/v1/research/events/world-monitor/inbox?limit=100"
    [pscustomobject]@{ mode = "read-only"; jaxHealth = $health.status; runtime = $runtime; maximumLeverage = $maxLeverage; safetyCounts = $before; inbox = $inbox } | ConvertTo-Json -Depth 12
    exit 0
}
if ($runtime.runtimeMode -ne "paper") { throw "Jax runtime is not paper mode: $($runtime.runtimeMode)" }
if ($runtime.allowLiveTrading) { throw "ALLOW_LIVE_TRADING is enabled" }
if ($runtime.executionEnabled) { throw "Broker execution is enabled" }
if ($runtime.executionInstructionWorkerEnabled) { throw "Execution worker is enabled" }
if ([double]$maxLeverage -gt 1) { throw "Maximum leverage exceeds 1x: $maxLeverage" }

$oldEndpoint = $env:JAX_RESEARCH_ENDPOINT_URL
$oldToken = $env:JAX_RESEARCH_TOKEN
try {
    $env:JAX_RESEARCH_ENDPOINT_URL = $JaxApiUrl
    $env:JAX_RESEARCH_TOKEN = $token
    Push-Location -LiteralPath $worldMonitorRoot
    try {
        $collectionJson = & npx tsx scripts/jax-live-ingestion.mts 2>&1 | Out-String
        if ($LASTEXITCODE -ne 0) { throw "World Monitor collection failed: $collectionJson" }
    } finally {
        Pop-Location
    }
    $collection = $collectionJson | ConvertFrom-Json
} finally {
    $env:JAX_RESEARCH_ENDPOINT_URL = $oldEndpoint
    $env:JAX_RESEARCH_TOKEN = $oldToken
}
$after = Get-SafetyCounts
foreach ($field in @('automaticApprovals','executionInstructions','orderIntents','brokerOrders','trades','fills')) {
    if ([long]$after.$field -ne [long]$before.$field) { throw "Safety count changed for ${field}: $($before.$field) -> $($after.$field)" }
}

$inbox = Invoke-JaxGet "/api/v1/research/events/world-monitor/inbox?limit=100"
[pscustomobject]@{
    mode = "live-genuine-collection"; jaxHealth = $health.status; runtime = $runtime; maximumLeverage = $maxLeverage
    sourcesChecked = $collection.sourcesChecked; sourceItemsObserved = $collection.observed; eventsConsidered = $collection.considered
    filtered = $collection.filtered; delivered = $collection.delivered.Count; accepted = $collection.accepted
    rejected = $collection.rejected; deduplicated = $collection.deduplicated; deliveries = $collection.delivered
    safetyBefore = $before; safetyAfter = $after; inbox = $inbox
} | ConvertTo-Json -Depth 12
