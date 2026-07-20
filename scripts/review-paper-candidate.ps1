<#
.SYNOPSIS
Safely reviews and records an operator decision for one paper candidate.

.DESCRIPTION
Shows the complete candidate and risk review, checks approval eligibility and
expiry, then requires an explicit approve or reject confirmation. After the
decision it reloads the persisted candidate, approval, and paper-ticket state
and verifies that no execution instruction or broker order was created.

.PARAMETER CandidateId
The candidate identifier shown in the approval queue.

.PARAMETER ApiUrl
The local Jax trader API base URL.

.PARAMETER Operator
The operator name recorded with the decision. Defaults to the current user.

.PARAMETER ReviewOnly
Display and verify the candidate without offering to approve or reject it.

.EXAMPLE
.\scripts\review-paper-candidate.ps1 `
  -CandidateId "39cb350a-d796-4f7e-8c05-d746acb6b1d6"
#>
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true, Position = 0)]
    [ValidatePattern('^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$')]
    [string]$CandidateId,

    [ValidateNotNullOrEmpty()]
    [string]$ApiUrl = "http://localhost:8081",

    [ValidateNotNullOrEmpty()]
    [string]$Operator = $env:USERNAME,

    [switch]$ReviewOnly
)

$ErrorActionPreference = "Stop"
$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$ApiUrl = $ApiUrl.TrimEnd('/')
$script:ApiHeaders = @{ "X-User-ID" = $Operator }

function Get-ConfigValue([string]$Name) {
    $value = [Environment]::GetEnvironmentVariable($Name)
    if (-not [string]::IsNullOrWhiteSpace($value)) {
        return $value.Trim()
    }

    $envPath = Join-Path $RepoRoot ".env"
    if (-not (Test-Path -LiteralPath $envPath)) {
        return $null
    }

    $line = Get-Content -LiteralPath $envPath |
        Where-Object { $_ -match "^$([regex]::Escape($Name))=" } |
        Select-Object -First 1
    if ($null -eq $line) {
        return $null
    }
    return (($line -split "=", 2)[1]).Trim().Trim('"')
}

function Initialize-ApiAuth {
    try {
        $authStatus = Invoke-RestMethod -Method Get -Uri "$ApiUrl/auth/status" -TimeoutSec 10
    } catch {
        throw "Jax is not reachable at $ApiUrl. Start the local stack, then try again. $($_.Exception.Message)"
    }

    if (-not $authStatus.enabled) {
        return
    }

    $username = Get-ConfigValue "AUTH_BOOTSTRAP_USERNAME"
    $password = Get-ConfigValue "AUTH_BOOTSTRAP_PASSWORD"
    if ([string]::IsNullOrWhiteSpace($username) -or [string]::IsNullOrWhiteSpace($password)) {
        throw "Jax authentication is enabled, but the local operator credentials are not configured. Ask the system administrator to configure AUTH_BOOTSTRAP_USERNAME and AUTH_BOOTSTRAP_PASSWORD."
    }

    $loginBody = @{ username = $username; password = $password } | ConvertTo-Json
    $login = Invoke-RestMethod -Method Post -Uri "$ApiUrl/auth/login" -ContentType "application/json" -Body $loginBody -TimeoutSec 10
    if ([string]::IsNullOrWhiteSpace($login.access_token)) {
        throw "Jax login succeeded but did not return an access token."
    }
    $script:ApiHeaders.Authorization = "Bearer $($login.access_token)"
}

function Invoke-JaxGet([string]$Path, [switch]$AllowNotFound) {
    try {
        return Invoke-RestMethod -Method Get -Uri "$ApiUrl$Path" -Headers $script:ApiHeaders -TimeoutSec 15
    } catch {
        if ($AllowNotFound -and $null -ne $_.Exception.Response -and [int]$_.Exception.Response.StatusCode -eq 404) {
            return $null
        }
        $detail = if ($_.ErrorDetails.Message) { $_.ErrorDetails.Message.Trim() } else { $_.Exception.Message }
        throw "Jax could not load the candidate review: $detail"
    }
}

function Get-Candidate {
    return Invoke-JaxGet "/api/v1/candidates/$CandidateId"
}

function Get-ApprovalDetail {
    return Invoke-JaxGet "/api/v1/approvals/$CandidateId" -AllowNotFound
}

function Get-ReviewFacts([object]$Candidate, [object[]]$ApprovalQueue) {
    $metadata = $Candidate.metadata
    $riskReview = $metadata.riskReview.result
    $eligibility = $metadata.riskReview.approvalEligibility
    $expiresAt = if ($Candidate.expiresAt) { [DateTimeOffset]::Parse([string]$Candidate.expiresAt) } else { $null }
    $expired = $null -ne $expiresAt -and [DateTimeOffset]::UtcNow -gt $expiresAt
    $evidenceConfirmed = @($ApprovalQueue | Where-Object { [string]$_.id -eq $CandidateId }).Count -eq 1

    $eligible =
        $Candidate.status -eq "awaiting_approval" -and
        $Candidate.gateStatus -eq "ready_for_risk_review" -and
        $Candidate.riskStatus -eq "ready_for_approval_review" -and
        $Candidate.approvalStatus -eq "approval_review_ready" -and
        $Candidate.humanApprovalRequired -eq $true -and
        $eligibility.approvalEligible -eq $true -and
        $riskReview.riskReady -eq $true -and
        $evidenceConfirmed -and
        $metadata.paperOnly -eq $true -and
        -not $expired -and
        $null -eq $Candidate.latestApproval -and
        $null -eq $Candidate.executionInstructionId -and
        $null -eq $Candidate.tradeId

    [pscustomobject]@{
        ExpiresAt = $expiresAt
        Expired = $expired
        Eligible = $eligible
        EvidenceStatus = if ($evidenceConfirmed) { "sufficient" } else { "not confirmed in approval queue" }
        RiskReview = $riskReview
        Eligibility = $eligibility
    }
}

function Write-Review([object]$Candidate, [object]$Facts) {
    Write-Host ""
    Write-Host "Paper candidate review" -ForegroundColor Cyan
    Write-Host "  Candidate: $($Candidate.id)"
    Write-Host "  Symbol / direction: $($Candidate.symbol) / $($Candidate.direction)"
    Write-Host "  Setup: $($Candidate.setupType)"
    Write-Host "  Lifecycle: $($Candidate.status)"
    Write-Host "  Evidence: $($Facts.EvidenceStatus)"
    Write-Host "  Trust gate: $($Candidate.gateStatus)"
    Write-Host "  Risk: $($Candidate.riskStatus)"
    Write-Host "  Approval: $($Candidate.approvalStatus)"
    Write-Host "  Paper only: $($Candidate.metadata.paperOnly)"
    Write-Host "  Expires: $(if ($Facts.ExpiresAt) { $Facts.ExpiresAt.ToString('u') } else { 'not set' })"
    Write-Host "  Expired: $($Facts.Expired)" -ForegroundColor $(if ($Facts.Expired) { 'Red' } else { 'Green' })
    Write-Host "  Eligible for approval now: $($Facts.Eligible)" -ForegroundColor $(if ($Facts.Eligible) { 'Green' } else { 'Yellow' })

    Write-Host ""
    Write-Host "Risk review" -ForegroundColor Cyan
    $Facts.RiskReview | Format-List | Out-Host
    Write-Host "Approval eligibility" -ForegroundColor Cyan
    $Facts.Eligibility | Format-List | Out-Host
    Write-Host "Complete candidate record" -ForegroundColor Cyan
    $Candidate | ConvertTo-Json -Depth 30 | Write-Host
}

function Read-Decision {
    while ($true) {
        $choice = (Read-Host "Decision: type APPROVE, REJECT, or CANCEL").Trim().ToUpperInvariant()
        if ($choice -in @("APPROVE", "REJECT", "CANCEL")) {
            return $choice
        }
        Write-Host "Please type exactly APPROVE, REJECT, or CANCEL." -ForegroundColor Yellow
    }
}

function Submit-Decision([string]$Decision, [string]$Notes) {
    $action = $Decision.ToLowerInvariant()
    $body = @{ notes = $Notes } | ConvertTo-Json
    try {
        return Invoke-RestMethod -Method Post -Uri "$ApiUrl/api/v1/approvals/$CandidateId/$action" `
            -Headers $script:ApiHeaders -ContentType "application/json" -Body $body -TimeoutSec 30
    } catch {
        $detail = if ($_.ErrorDetails.Message) { $_.ErrorDetails.Message.Trim() } else { $_.Exception.Message }
        throw "The $action decision was not accepted: $detail"
    }
}

function Assert-PersistedDecision([string]$ExpectedDecision, [object]$Candidate, [object]$Detail) {
    if ($null -eq $Detail -or $null -eq $Detail.latestApproval) {
        throw "Verification failed: no persisted approval decision was returned."
    }
    if ($Detail.latestApproval.decision -ne $ExpectedDecision -or $Candidate.latestApproval.decision -ne $ExpectedDecision) {
        throw "Verification failed: the persisted decision does not match '$ExpectedDecision'."
    }
    if ($Detail.latestApproval.approvedBy -ne $Operator) {
        throw "Verification failed: the persisted operator is '$($Detail.latestApproval.approvedBy)', expected '$Operator'."
    }
}

function Assert-NoExecution([object]$Candidate, [object]$Detail) {
    $ticket = $Detail.paperTicket
    $unsafeTicket = $null -ne $ticket -and (
        $ticket.paperOnly -ne $true -or
        $ticket.brokerExecutionAllowed -eq $true -or
        $ticket.executionInstructionCreated -eq $true -or
        $ticket.liveTradingAllowed -eq $true
    )
    if ($null -ne $Candidate.executionInstructionId -or $null -ne $Candidate.tradeId -or $null -ne $Detail.execution -or $unsafeTicket) {
        throw "SAFETY VERIFICATION FAILED: an execution instruction, broker-linked trade, or unsafe paper ticket is present. Stop and investigate before continuing."
    }
}

function Write-PersistedResult([object]$Detail) {
    Write-Host ""
    Write-Host "Persisted decision" -ForegroundColor Green
    Write-Host "  Verification state: $($Detail.state)"
    Write-Host "  Decision: $($Detail.latestApproval.decision)"
    Write-Host "  Operator: $($Detail.latestApproval.approvedBy)"
    Write-Host "  Reason: $($Detail.latestApproval.notes)"
    Write-Host "  Decided at: $($Detail.latestApproval.decidedAt)"
    if ($null -ne $Detail.paperTicket) {
        Write-Host "  Paper ticket: $($Detail.paperTicket.paperTicketId)"
        Write-Host "  Paper-ticket state: $($Detail.paperTicket.status)"
        Write-Host "  Paper only: $($Detail.paperTicket.paperOnly)"
    } else {
        Write-Host "  Paper ticket: not created"
    }
    Write-Host "  Execution instruction: not created" -ForegroundColor Green
    Write-Host "  Broker order: not created" -ForegroundColor Green
}

Initialize-ApiAuth
$candidate = Get-Candidate
if ([string]$candidate.id -ne $CandidateId) {
    throw "Jax returned a different candidate than requested."
}
$approvalQueue = @(Invoke-JaxGet "/api/v1/approvals/queue?limit=200")
$facts = Get-ReviewFacts $candidate $approvalQueue
Write-Review $candidate $facts

$existingDetail = Get-ApprovalDetail
Assert-NoExecution $candidate $existingDetail
Write-Host "Initial safety check: no execution instruction or broker order is associated with this candidate." -ForegroundColor Green

if ($null -ne $existingDetail -and $null -ne $existingDetail.latestApproval) {
    Write-PersistedResult $existingDetail
    Write-Host "An existing decision is already persisted. No new decision will be requested or submitted." -ForegroundColor Cyan
    return
}

if ($ReviewOnly) {
    Write-Host "Review-only mode: no decision was submitted." -ForegroundColor Cyan
    return
}

$decision = Read-Decision
if ($decision -eq "CANCEL") {
    Write-Host "Cancelled. No decision was submitted." -ForegroundColor Yellow
    return
}
if ($decision -eq "APPROVE" -and -not $facts.Eligible) {
    throw "This candidate is not eligible for approval. No decision was submitted. It may be expired or may no longer satisfy the evidence, trust, risk, or paper-only gates."
}

$notes = (Read-Host "Enter the reason for this $($decision.ToLowerInvariant()) decision").Trim()
if ([string]::IsNullOrWhiteSpace($notes)) {
    throw "A decision reason is required. No decision was submitted."
}
$confirmation = (Read-Host "To persist the decision, type '$decision $CandidateId'").Trim()
if ($confirmation -cne "$decision $CandidateId") {
    throw "Confirmation did not match. No decision was submitted."
}

$immediateResponse = Submit-Decision $decision $notes
Write-Host "Immediate decision API response" -ForegroundColor Cyan
$immediateResponse | ConvertTo-Json -Depth 30 | Write-Host
$expectedDecision = if ($decision -eq "APPROVE") { "approved" } else { "rejected" }
try {
    $persistedCandidate = Get-Candidate
    $persistedDetail = Get-ApprovalDetail
    Assert-PersistedDecision $expectedDecision $persistedCandidate $persistedDetail
    Assert-NoExecution $persistedCandidate $persistedDetail
} catch {
    Write-Host "The decision request may already have succeeded. Do not submit it again." -ForegroundColor Red
    Write-Host "Candidate diagnostic identifier: $CandidateId" -ForegroundColor Yellow
    Write-Host "Read-only diagnostic command: .\scripts\review-paper-candidate.ps1 -CandidateId `"$CandidateId`" -ReviewOnly" -ForegroundColor Yellow
    throw "Post-decision verification failed after the API accepted the request: $($_.Exception.Message)"
}

Write-Host "Decision persisted and verified" -ForegroundColor Green
Write-PersistedResult $persistedDetail
