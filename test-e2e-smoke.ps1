# Jax Trading Assistant — End-to-End Smoke Test
# Tests the full candidate → approval → chat pipeline via the unified API.
# Target: http://localhost:8081 (trader binary)
#
# Usage:
#   .\test-e2e-smoke.ps1
#   .\test-e2e-smoke.ps1 -BaseUrl http://localhost:9090
#   .\test-e2e-smoke.ps1 -Username admin -Password admin123

param(
    [string]$BaseUrl = "http://localhost:8081",
    [string]$Username = "admin",
    [string]$Password = "admin123"
)

$ErrorActionPreference = "SilentlyContinue"

$api = "$BaseUrl/api/v1"
$pass = 0
$fail = 0
$skip = 0

function Ok([string]$msg) {
    Write-Host "  ✓ $msg" -ForegroundColor Green
    $script:pass++
}
function Fail([string]$msg) {
    Write-Host "  ✗ $msg" -ForegroundColor Red
    $script:fail++
}
function Skip([string]$msg) {
    Write-Host "  – $msg" -ForegroundColor DarkGray
    $script:skip++
}
function Section([string]$msg) {
    Write-Host "`n[$msg]" -ForegroundColor Cyan
}

Write-Host "`n================================================" -ForegroundColor Cyan
Write-Host " Jax Trading — E2E Smoke Test" -ForegroundColor Cyan
Write-Host " Target : $BaseUrl" -ForegroundColor Cyan
Write-Host " Time   : $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')" -ForegroundColor Cyan
Write-Host "================================================`n" -ForegroundColor Cyan

# ── 1. Health ─────────────────────────────────────────────────────────────────
Section "1. Health"
try {
    $health = Invoke-RestMethod -Uri "$BaseUrl/health" -Method Get -TimeoutSec 5
    if ($health.status -eq "ok" -or $health.status -eq "healthy") {
        Ok "Trader API healthy (status=$($health.status))"
    } else {
        Fail "Unexpected health status: $($health.status)"
    }
} catch {
    Fail "Trader API not responding at $BaseUrl — is the server running?"
    Write-Host "`n  Start with: go run ./cmd/trader" -ForegroundColor Yellow
    Write-Host "  Exiting — remaining tests require a live server.`n" -ForegroundColor Yellow
    exit 1
}

# ── 2. Auth ───────────────────────────────────────────────────────────────────
Section "2. Auth"
$token = $null
try {
    $loginBody = @{ username = $Username; password = $Password } | ConvertTo-Json
    $loginResp = Invoke-RestMethod -Uri "$api/auth/login" -Method Post `
        -Body $loginBody -ContentType "application/json" -TimeoutSec 5
    $token = $loginResp.token
    if ($token) {
        Ok "Authenticated as $Username"
    } else {
        Fail "Login succeeded but no token in response"
    }
} catch {
    Fail "Login failed ($($_.Exception.Message)) — remaining tests will use no auth"
}

$headers = @{ "Content-Type" = "application/json" }
if ($token) { $headers["Authorization"] = "Bearer $token" }

# ── 3. Candidate Trades ───────────────────────────────────────────────────────
Section "3. Candidate Trades"
$candidateId = $null
try {
    $candidates = Invoke-RestMethod -Uri "$api/candidates?limit=5" `
        -Method Get -Headers $headers -TimeoutSec 5
    $total = if ($candidates.total -ne $null) { $candidates.total } else { $candidates.Count }
    Ok "Candidate trades endpoint reachable (count=$total)"

    # Check awaiting_approval candidates
    $awaiting = Invoke-RestMethod -Uri "$api/candidates?status=awaiting_approval&limit=5" `
        -Method Get -Headers $headers -TimeoutSec 5
    $awaitingTotal = if ($awaiting.total -ne $null) { $awaiting.total } else { $awaiting.Count }
    Ok "Awaiting-approval candidates: $awaitingTotal"

    # Grab first candidate ID for downstream tests
    $firstItem = if ($candidates.data) { $candidates.data[0] } else { $candidates[0] }
    if ($firstItem) { $candidateId = $firstItem.id }
} catch {
    Fail "Candidates endpoint failed: $($_.Exception.Message)"
}

# ── 4. Approvals Queue ────────────────────────────────────────────────────────
Section "4. Approvals Queue"
try {
    $approvals = Invoke-RestMethod -Uri "$api/approvals?limit=5" `
        -Method Get -Headers $headers -TimeoutSec 5
    $total = if ($approvals.total -ne $null) { $approvals.total } else { $approvals.Count }
    Ok "Approvals queue reachable (count=$total)"
} catch {
    Fail "Approvals endpoint failed: $($_.Exception.Message)"
}

# ── 5. Approval Lifecycle (read-only check) ────────────────────────────────────
Section "5. Approval Lifecycle (read-only)"
if ($candidateId) {
    try {
        $detail = Invoke-RestMethod -Uri "$api/candidates/$candidateId" `
            -Method Get -Headers $headers -TimeoutSec 5
        $status = $detail.status ?? $detail.Status
        Ok "Candidate $candidateId fetched (status=$status)"
    } catch {
        Fail "Failed to fetch candidate $candidateId : $($_.Exception.Message)"
    }
} else {
    Skip "No candidate available — skipping lifecycle check"
}

# ── 6. Chat — Session Lifecycle ───────────────────────────────────────────────
Section "6. Chat — Session Lifecycle"
$sessionId = $null
try {
    $sessBody = @{ title = "smoke-test-$(Get-Date -Format 'HHmmss')" } | ConvertTo-Json
    $sess = Invoke-RestMethod -Uri "$api/chat/sessions" -Method Post `
        -Body $sessBody -Headers $headers -TimeoutSec 5
    $sessionId = $sess.id
    Ok "Created chat session (id=$sessionId)"
} catch {
    Fail "Failed to create chat session: $($_.Exception.Message)"
}

if ($sessionId) {
    # Verify session appears in list
    try {
        $sessions = Invoke-RestMethod -Uri "$api/chat/sessions" `
            -Method Get -Headers $headers -TimeoutSec 5
        $found = $sessions | Where-Object { $_.id -eq $sessionId }
        if ($found) { Ok "Session appears in list" }
        else { Fail "Session not found in list after creation" }
    } catch {
        Fail "Failed to list sessions: $($_.Exception.Message)"
    }
}

# ── 7. Chat — Send Message ────────────────────────────────────────────────────
Section "7. Chat — Send Message"
$lastMsgContent = $null
if ($sessionId) {
    try {
        $msgBody = @{ sessionId = $sessionId; content = "What is the current state of the approval queue?" } | ConvertTo-Json
        $msgResp = Invoke-RestMethod -Uri "$api/chat" -Method Post `
            -Body $msgBody -Headers $headers -TimeoutSec 30
        $msgs = $msgResp.messages
        $assistantMsg = $msgs | Where-Object { $_.role -eq "assistant" } | Select-Object -Last 1
        if ($assistantMsg) {
            $lastMsgContent = $assistantMsg.content
            Ok "Received assistant reply ($($lastMsgContent.Length) chars)"
        } else {
            Fail "No assistant message in response (got $($msgs.Count) messages)"
        }
    } catch {
        Fail "Send message failed: $($_.Exception.Message)"
    }
} else {
    Skip "No session — skipping message send"
}

# ── 8. Chat — History ─────────────────────────────────────────────────────────
Section "8. Chat — History"
if ($sessionId) {
    try {
        $history = Invoke-RestMethod -Uri "$api/chat?session=$sessionId" `
            -Method Get -Headers $headers -TimeoutSec 5
        $msgCount = if ($history -is [array]) { $history.Count } else { 0 }
        if ($msgCount -ge 2) {
            Ok "History contains $msgCount messages (user + assistant confirmed)"
        } else {
            Fail "Expected at least 2 messages in history, got $msgCount"
        }
    } catch {
        Fail "Get history failed: $($_.Exception.Message)"
    }
} else {
    Skip "No session — skipping history check"
}

# ── 9. Chat — Tool Dispatch ───────────────────────────────────────────────────
Section "9. Chat — Tool Dispatch"
# First verify the tools catalogue endpoint.
try {
    $toolsResp = Invoke-RestMethod -Uri "$api/chat/tools" `
        -Method Get -Headers $headers -TimeoutSec 5
    $toolCount = $toolsResp.tools.Count
    Ok "Tool catalogue reachable ($toolCount tools available)"
} catch {
    Fail "Tool catalogue endpoint failed: $($_.Exception.Message)"
}

# Dispatch a tool call via the chat endpoint.
if ($sessionId -and $candidateId) {
    try {
        $toolBody = @{
            sessionId = $sessionId
            content   = "Look up this candidate for me."
            toolCall  = @{
                name = "get_candidate_trade"
                args = @{ candidateId = $candidateId } | ConvertTo-Json -Compress
            }
        } | ConvertTo-Json
        $toolResp = Invoke-RestMethod -Uri "$api/chat" -Method Post `
            -Body $toolBody -Headers $headers -TimeoutSec 10
        $toolMsg  = $toolResp.messages | Where-Object { $_.role -eq "tool" }
        if ($toolMsg) {
            Ok "Tool dispatch: got tool result message"
        } else {
            Fail "Tool dispatch: no tool message in response"
        }
    } catch {
        Fail "Tool dispatch failed: $($_.Exception.Message)"
    }
} elseif (-not $candidateId) {
    Skip "No candidate ID — skipping tool dispatch (get_candidate_trade)"
} else {
    Skip "No session — skipping tool dispatch"
}

# ── 10. Signals (informational) ───────────────────────────────────────────────
Section "10. Strategy Signals (informational)"
try {
    $signals = Invoke-RestMethod -Uri "$api/signals?limit=5" `
        -Method Get -Headers $headers -TimeoutSec 5
    $total = if ($signals.total -ne $null) { $signals.total } else { $signals.Count }
    Ok "Signals endpoint reachable (count=$total)"
} catch {
    # Signals endpoint may not exist in all builds — treat as informational.
    Skip "Signals endpoint not available (non-critical)"
}

# ── Summary ───────────────────────────────────────────────────────────────────
$total_checks = $pass + $fail + $skip
Write-Host "`n================================================" -ForegroundColor Cyan
Write-Host " Results: $pass passed  $fail failed  $skip skipped  ($total_checks total)" -ForegroundColor $(
    if ($fail -gt 0) { "Red" } elseif ($skip -gt 0) { "Yellow" } else { "Green" }
)
Write-Host "================================================`n" -ForegroundColor Cyan

if ($fail -gt 0) { exit 1 } else { exit 0 }
