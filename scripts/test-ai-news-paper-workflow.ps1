param(
    [string]$ApiUrl = "http://localhost:8081",
    [string]$FrontendUrl = "http://localhost:5173",
    [switch]$SkipStackStart,
    [switch]$SkipBrowserE2E
)

$ErrorActionPreference = "Stop"

$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$script:ApiHeaders = @{}

function Get-ConfigValue([string]$Name) {
    $value = [Environment]::GetEnvironmentVariable($Name)
    if (-not [string]::IsNullOrWhiteSpace($value)) {
        return $value.Trim()
    }

    $envPath = Join-Path $RepoRoot ".env"
    if (-not (Test-Path -LiteralPath $envPath)) {
        return $null
    }

    $line = Get-Content -LiteralPath $envPath | Where-Object { $_ -match "^$([regex]::Escape($Name))=" } | Select-Object -First 1
    if ($null -eq $line) {
        return $null
    }

    return (($line -split "=", 2)[1]).Trim().Trim('"')
}

function Test-HttpOk([string]$Url) {
    try {
        $response = Invoke-WebRequest -Uri $Url -TimeoutSec 3 -UseBasicParsing -ErrorAction Stop
        return ($response.StatusCode -ge 200 -and $response.StatusCode -lt 500)
    } catch {
        return $false
    }
}

function Wait-ForHttp([string]$Url, [int]$Attempts = 30, [int]$DelaySeconds = 2) {
    for ($i = 1; $i -le $Attempts; $i++) {
        if (Test-HttpOk $Url) {
            return
        }
        Start-Sleep -Seconds $DelaySeconds
    }

    throw "Timed out waiting for $Url"
}

function Invoke-JsonPost([string]$Url, [object]$Body) {
    $json = $Body | ConvertTo-Json -Depth 12
    return Invoke-RestMethod -Method Post -Uri $Url -Headers $script:ApiHeaders -Body $json -ContentType "application/json" -TimeoutSec 30
}

function Initialize-ApiAuth([string]$BaseUrl) {
    $script:ApiHeaders = @{}

    $authStatus = Invoke-RestMethod -Method Get -Uri "$BaseUrl/auth/status" -TimeoutSec 10
    if (-not $authStatus.enabled) {
        Write-Host "  Auth disabled for API smoke." -ForegroundColor Green
        return
    }

    $username = Get-ConfigValue "AUTH_BOOTSTRAP_USERNAME"
    $password = Get-ConfigValue "AUTH_BOOTSTRAP_PASSWORD"
    if ([string]::IsNullOrWhiteSpace($username) -or [string]::IsNullOrWhiteSpace($password)) {
        throw "Auth is enabled, but AUTH_BOOTSTRAP_USERNAME/AUTH_BOOTSTRAP_PASSWORD are not configured in the environment or .env."
    }

    $body = @{ username = $username; password = $password } | ConvertTo-Json
    $login = Invoke-RestMethod -Method Post -Uri "$BaseUrl/auth/login" -ContentType "application/json" -Body $body -TimeoutSec 10
    if ([string]::IsNullOrWhiteSpace($login.access_token)) {
        throw "Auth login succeeded but no access_token was returned."
    }

    $script:ApiHeaders = @{ Authorization = "Bearer $($login.access_token)" }
    Write-Host "  Auth enabled; bootstrap login succeeded." -ForegroundColor Green
}

Push-Location $RepoRoot
try {
    Write-Host "AI/news paper workflow smoke" -ForegroundColor Cyan

    if (-not (Test-HttpOk "$ApiUrl/health")) {
        if ($SkipStackStart.IsPresent) {
            throw "Trader frontend API is not healthy at $ApiUrl/health and -SkipStackStart was supplied."
        }

        Write-Host "  Backend not ready. Starting local stack with start.ps1 -NoBrowser..." -ForegroundColor Yellow
        & powershell -NoProfile -ExecutionPolicy Bypass -File ".\start.ps1" -NoBrowser
    }

    Wait-ForHttp "$ApiUrl/health" 45 2
    Write-Host "  Trader frontend API healthy at $ApiUrl" -ForegroundColor Green
    Initialize-ApiAuth $ApiUrl

    if (-not (Test-HttpOk "$FrontendUrl/")) {
        Write-Host "  Frontend not reachable at $FrontendUrl. start.ps1 should have launched it; waiting..." -ForegroundColor Yellow
        Wait-ForHttp "$FrontendUrl/" 30 2
    }
    Write-Host "  Frontend ready at $FrontendUrl" -ForegroundColor Green

    Write-Host "  Cleaning fixture macro rows..." -ForegroundColor Gray
    & powershell -NoProfile -ExecutionPolicy Bypass -File ".\scripts\clean-fixture-runtime-data.ps1"
    if ($LASTEXITCODE -ne 0) {
        throw "Fixture cleanup failed."
    }
    Write-Host "  Fixture macro rows cleaned." -ForegroundColor Green

    Write-Host "  Syncing dataset snapshots..." -ForegroundColor Gray
    & powershell -NoProfile -ExecutionPolicy Bypass -File ".\scripts\sync-dataset-snapshots.ps1"
    if ($LASTEXITCODE -ne 0) {
        throw "Dataset snapshot sync failed."
    }

    $sourceEventId = "workflow-smoke-$([Guid]::NewGuid().ToString('N'))"
    $trigger = @{
        source = "world-monitor"
        source_event_id = $sourceEventId
        event_type = "macro_rates"
        headline = "Workflow smoke: softer inflation supports QQQ review"
        summary = "Synthetic operator smoke trigger used to verify world news promotion into the paper approval workflow."
        source_urls = @(
            "https://example.com/jax-workflow-smoke/source-1",
            "https://example.com/jax-workflow-smoke/source-2"
        )
        source_count = 2
        timestamp_utc = (Get-Date).ToUniversalTime().AddMinutes(-5).ToString("o")
        region = "US"
        possible_affected_etfs = @("QQQ")
        asset_themes = @("rates", "growth_equities")
        severity = "high"
        source_tier = "tier2"
        confidence = 0.74
        confidence_reasons = @("two independent sources", "macro/rates category", "mapped to QQQ")
        reason = "Rates-sensitive news can affect growth ETF positioning."
        raw_payload = @{
            fixture = $true
            workflowSmoke = $true
        }
    }

    Write-Host "  Posting World News Monitor trigger $sourceEventId..." -ForegroundColor Gray
    $ingest = Invoke-JsonPost "$ApiUrl/api/v1/research/events/world-monitor" $trigger
    Write-Host "  Trigger accepted: inbox=$($ingest.inbox_id) status=$($ingest.status)" -ForegroundColor Green

    $monitorStatus = Invoke-RestMethod -Method Get -Uri "$ApiUrl/api/v1/research/events/world-monitor/status" -Headers $script:ApiHeaders -TimeoutSec 15
    if ($monitorStatus.lastSourceEventId -ne $sourceEventId) {
        throw "World Monitor status did not reflect the posted trigger. Expected $sourceEventId, got $($monitorStatus.lastSourceEventId)."
    }
    Write-Host "  Monitor status: last=$($monitorStatus.lastStatus), headline='$($monitorStatus.lastHeadline)'" -ForegroundColor Green

    Write-Host "  Running promoter once..." -ForegroundColor Gray
    $promotion = Invoke-RestMethod -Method Post -Uri "$ApiUrl/api/v1/research/events/world-monitor/promote" -Headers $script:ApiHeaders -TimeoutSec 30
    $promotedCount = @($promotion.promoted).Count
    Write-Host "  Promoted $promotedCount trigger(s); skipped=$($promotion.skipped)" -ForegroundColor Green

    if ($promotedCount -lt 1) {
        throw "World Monitor trigger did not promote into an AI opportunity. Check market quote availability and scanner settings."
    }

    $promoted = @($promotion.promoted)[0]
    if ($promoted.route -ne "approval_required") {
        throw "Promoted trigger route is '$($promoted.route)', not approval_required. The chart gate or policy gate likely blocked the candidate; review AI Trading evidence for candidate $($promoted.candidateId)."
    }

    Write-Host "  Approving paper candidate $($promoted.candidateId)..." -ForegroundColor Gray
    $approvalDetail = Invoke-JsonPost "$ApiUrl/api/v1/approvals/$($promoted.candidateId)/approve" @{
        notes = "Operator smoke approval from scripts/test-ai-news-paper-workflow.ps1"
    }
    if ([string]::IsNullOrWhiteSpace($approvalDetail.execution.id)) {
        throw "Approval succeeded but no execution instruction was returned for candidate $($promoted.candidateId)."
    }
    Write-Host "  Execution instruction created: $($approvalDetail.execution.id) status=$($approvalDetail.execution.status)" -ForegroundColor Green

    $pilotStatus = Invoke-RestMethod -Method Get -Uri "$ApiUrl/api/v1/trading/pilot-status" -Headers $script:ApiHeaders -TimeoutSec 15
    Write-Host "  Broker readiness: connected=$($pilotStatus.brokerConnected), readOnly=$($pilotStatus.readOnly), paper=$($pilotStatus.paperTrading)" -ForegroundColor Green

    $overview = Invoke-RestMethod -Method Get -Uri "$ApiUrl/api/v1/ai/overview" -Headers $script:ApiHeaders -TimeoutSec 15
    $counts = $overview.opportunityCounts
    $total = [int]$counts.signalsPending + [int]$counts.candidates + [int]$counts.approvals
    Write-Host "  AI Trading counts: signals=$($counts.signalsPending), candidates=$($counts.candidates), approvals=$($counts.approvals)" -ForegroundColor Green

    if ($total -lt 1) {
        throw "AI Trading overview reported zero opportunities after promotion."
    }

    if (-not $SkipBrowserE2E.IsPresent) {
        Write-Host "  Running browser workflow E2E..." -ForegroundColor Gray
        Push-Location "frontend"
        try {
            & npm run test:e2e -- --grep "ai news paper workflow"
            if ($LASTEXITCODE -ne 0) {
                throw "Playwright AI/news paper workflow E2E failed."
            }
        } finally {
            Pop-Location
        }
        Write-Host "  Browser E2E approved one paper opportunity and created a paper instruction." -ForegroundColor Green
    }

    Write-Host "AI/news paper workflow smoke passed. No live-trading path was used." -ForegroundColor Green
} finally {
    Pop-Location
}
