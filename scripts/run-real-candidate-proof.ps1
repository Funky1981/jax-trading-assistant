param(
  [string]$ApiBase = "http://localhost:8081",
  [string]$DatabaseUrl = $env:DATABASE_URL,
  [string]$OutputDir = "Docs/runs/real-candidate-proof"
)

$ErrorActionPreference = "Stop"
$requiredMigration = 46
$startedAt = (Get-Date).ToUniversalTime()
$checks = [System.Collections.Generic.List[object]]::new()
$headers = @{}

function Add-Check([string]$Name, [string]$Status, [string]$Detail) {
  $checks.Add([pscustomobject]@{ name = $Name; status = $Status; detail = $Detail })
  $color = if ($Status -eq "PASS") { "Green" } elseif ($Status -eq "WARN") { "Yellow" } else { "Red" }
  Write-Host ("[{0}] {1} - {2}" -f $Status, $Name, $Detail) -ForegroundColor $color
}

function Get-ConfigValue([string]$Name) {
  $value = [Environment]::GetEnvironmentVariable($Name)
  if (-not [string]::IsNullOrWhiteSpace($value)) { return $value.Trim() }
  if (-not (Test-Path ".env")) { return $null }
  $line = Get-Content ".env" | Where-Object { $_ -match "^$([regex]::Escape($Name))=" } | Select-Object -First 1
  if ($null -eq $line) { return $null }
  return (($line -split "=", 2)[1]).Trim().Trim('"')
}

function Invoke-SqlJson([string]$Sql) {
  $raw = & psql $DatabaseUrl --no-psqlrc -X -t -A -v ON_ERROR_STOP=1 -c $Sql 2>&1
  if ($LASTEXITCODE -ne 0) { throw ($raw -join [Environment]::NewLine) }
  $text = ($raw -join "").Trim()
  if ([string]::IsNullOrWhiteSpace($text)) { return $null }
  return $text | ConvertFrom-Json
}

function Initialize-Auth {
  try {
    $status = Invoke-RestMethod -Method GET -Uri "$ApiBase/auth/status" -TimeoutSec 10
    if (-not $status.enabled) { Add-Check "api-auth" "PASS" "authentication disabled"; return }
    $username = Get-ConfigValue "AUTH_BOOTSTRAP_USERNAME"
    $password = Get-ConfigValue "AUTH_BOOTSTRAP_PASSWORD"
    if ([string]::IsNullOrWhiteSpace($username) -or [string]::IsNullOrWhiteSpace($password)) {
      throw "AUTH_BOOTSTRAP_USERNAME and AUTH_BOOTSTRAP_PASSWORD are required"
    }
    $login = Invoke-RestMethod -Method POST -Uri "$ApiBase/auth/login" -ContentType "application/json" -Body (@{ username = $username; password = $password } | ConvertTo-Json) -TimeoutSec 10
    $script:headers = @{ Authorization = "Bearer $($login.access_token)" }
    Add-Check "api-auth" "PASS" "bootstrap login succeeded"
  } catch { throw "API authentication failed: $($_.Exception.Message)" }
}

if ([string]::IsNullOrWhiteSpace($DatabaseUrl)) { $DatabaseUrl = Get-ConfigValue "DATABASE_URL" }
if ([string]::IsNullOrWhiteSpace($DatabaseUrl)) { throw "DATABASE_URL is required" }
if (-not (Get-Command psql -ErrorAction SilentlyContinue)) { throw "psql is required on PATH" }

try {
  $db = Invoke-SqlJson "SELECT json_build_object('reachable', true, 'migrationVersion', version, 'dirty', dirty) FROM schema_migrations ORDER BY version DESC LIMIT 1;"
  Add-Check "postgres" "PASS" "reachable"
  if ($db.migrationVersion -eq $requiredMigration -and -not $db.dirty) {
    Add-Check "migrations" "PASS" "schema version $requiredMigration is clean"
  } else {
    Add-Check "migrations" "FAIL" "expected clean version $requiredMigration; found version $($db.migrationVersion), dirty=$($db.dirty)"
  }
} catch {
  Add-Check "postgres" "FAIL" $_.Exception.Message
  throw
}

$inputs = Invoke-SqlJson @"
SELECT json_build_object(
  'quotes', (SELECT COUNT(*) FROM quotes WHERE price > 0 OR (bid > 0 AND ask > 0)),
  'candles', (SELECT COUNT(*) FROM candles),
  'promotableInboxRows', (
    SELECT COUNT(*) FROM world_monitor_research_inbox
    WHERE status='new' AND candidate_id IS NULL AND normalized_event_id IS NOT NULL AND confidence >= 0.55
  ),
  'promotableSymbols', COALESCE((
    SELECT json_agg(DISTINCT etf.value)
    FROM world_monitor_research_inbox w
    CROSS JOIN LATERAL jsonb_array_elements_text(w.possible_affected_etfs) etf(value)
    WHERE w.status='new' AND w.candidate_id IS NULL AND w.normalized_event_id IS NOT NULL AND w.confidence >= 0.55
  ), '[]'::json)
);
"@

Add-Check "quotes" $(if ($inputs.quotes -gt 0) { "PASS" } else { "FAIL" }) "$($inputs.quotes) usable rows"
Add-Check "candles" $(if ($inputs.candles -gt 0) { "PASS" } else { "FAIL" }) "$($inputs.candles) rows"
Add-Check "world-monitor-inbox" $(if ($inputs.promotableInboxRows -gt 0) { "PASS" } else { "FAIL" }) "$($inputs.promotableInboxRows) promotable rows"

$promotion = [pscustomobject]@{ promoted = @(); skipped = 0; error = $null }
if (@($checks | Where-Object status -eq "FAIL").Count -eq 0) {
  Initialize-Auth
  try {
    $promotion = Invoke-RestMethod -Method POST -Uri "$ApiBase/api/v1/research/events/world-monitor/promote" -Headers $headers -ContentType "application/json" -Body "{}" -TimeoutSec 60
    Add-Check "promotion" "PASS" "$(@($promotion.promoted).Count) candidates created; $($promotion.skipped) skipped"
  } catch {
    $promotion = [pscustomobject]@{ promoted = @(); skipped = 0; error = $_.Exception.Message }
    Add-Check "promotion" "FAIL" $_.Exception.Message
  }
} else {
  Add-Check "promotion" "WARN" "not run because prerequisites failed"
}

$candidateIds = @($promotion.promoted | ForEach-Object { $_.candidateId } | Where-Object { $_ })
$candidateIdCsv = $candidateIds -join ","
$candidateFilter = if ($candidateIds.Count -gt 0) { "ct.id = ANY(string_to_array('$candidateIdCsv', ',')::uuid[])" } else { "FALSE" }

$candidates = Invoke-SqlJson @"
SELECT COALESCE(json_agg(row_to_json(report) ORDER BY report.created_at), '[]'::json)
FROM (
  SELECT ct.created_at,
         ct.id::text AS candidate_id,
         COALESCE(NULLIF(ct.source,''), ct.data_provenance) AS input_source,
         ct.symbol,
         COALESCE(NULLIF(ct.catalyst_summary,''), ct.metadata->'worldMonitor'->>'summary', ct.metadata->'worldMonitor'->>'headline', ct.reasoning, '') AS catalyst_summary,
         ct.entry_price::float8 AS entry,
         ct.stop_loss::float8 AS stop,
         ct.take_profit::float8 AS target,
         COALESCE(es.evidence_status, 'missing') AS evidence_status,
         ct.gate_status,
         ct.risk_status,
         ct.approval_status,
         COALESCE(pt.status, 'not_created') AS paper_ticket_status,
         ct.status AS candidate_status,
         COALESCE(ct.reject_reasons, '{}') || CASE WHEN NULLIF(ct.block_reason,'') IS NULL THEN '{}'::text[] ELSE ARRAY[ct.block_reason] END AS reject_reasons,
         COALESCE(pt.warning_reasons, '{}') AS warning_reasons,
         COALESCE(pt.paper_only, true) AS paper_only,
         COALESCE(pt.broker_execution_allowed, false) AS broker_execution_allowed,
         COALESCE(pt.execution_instruction_created, false) AS execution_instruction_created,
         COALESCE(pt.live_trading_allowed, false) AS live_trading_allowed,
         COALESCE(pt.leverage_allowed, false) AS leverage_allowed
  FROM candidate_trades ct
  LEFT JOIN LATERAL (
    SELECT evidence_status FROM candidate_evidence_scores WHERE candidate_id=ct.id ORDER BY scored_at DESC LIMIT 1
  ) es ON true
  LEFT JOIN candidate_paper_tickets pt ON pt.candidate_id=ct.id
  WHERE $candidateFilter
) report;
"@

$safety = Invoke-SqlJson @"
SELECT json_build_object(
  'executionInstructions', (SELECT COUNT(*) FROM execution_instructions ei WHERE ei.candidate_id = ANY(COALESCE(string_to_array(NULLIF('$candidateIdCsv',''), ',')::uuid[], ARRAY[]::uuid[]))),
  'unsafeTickets', (SELECT COUNT(*) FROM candidate_paper_tickets pt WHERE pt.candidate_id = ANY(COALESCE(string_to_array(NULLIF('$candidateIdCsv',''), ',')::uuid[], ARRAY[]::uuid[])) AND (NOT paper_only OR broker_execution_allowed OR execution_instruction_created OR live_trading_allowed OR leverage_allowed)),
  'leveragedCandidates', (SELECT COUNT(*) FROM candidate_trades ct WHERE ct.id = ANY(COALESCE(string_to_array(NULLIF('$candidateIdCsv',''), ',')::uuid[], ARRAY[]::uuid[])) AND COALESCE((ct.metadata->>'requestedLeverage')::numeric, 1) > 1)
);
"@

$safe = ($safety.executionInstructions -eq 0 -and $safety.unsafeTickets -eq 0 -and $safety.leveragedCandidates -eq 0)
Add-Check "safety-paper-only" $(if ($safe) { "PASS" } else { "FAIL" }) "execution instructions=$($safety.executionInstructions), unsafe tickets=$($safety.unsafeTickets), leveraged candidates=$($safety.leveragedCandidates)"

$ticketCount = @($candidates | Where-Object paper_ticket_status -ne "not_created").Count
$blockedCount = @($candidates | Where-Object candidate_status -eq "blocked").Count
$reviewableCount = @($candidates | Where-Object { $_.candidate_status -eq "awaiting_approval" -or $_.paper_ticket_status -ne "not_created" }).Count
$status = if (@($checks | Where-Object status -eq "FAIL").Count -gt 0) { "blocked" } elseif ($reviewableCount -gt 0) { "candidate_produced" } else { "no_candidate" }

$report = [pscustomobject]@{
  status = $status
  generatedAt = (Get-Date).ToUniversalTime().ToString("o")
  startedAt = $startedAt.ToString("o")
  inputs = $inputs
  promotion = $promotion
  summary = [pscustomobject]@{ candidatesCreated = $candidateIds.Count; reviewableCandidates = $reviewableCount; blockedCandidates = $blockedCount; skippedCandidates = [int]$promotion.skipped; paperTicketsCreated = $ticketCount }
  candidates = @($candidates)
  safety = $safety
  checks = $checks
}

New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null
$stamp = Get-Date -Format "yyyyMMdd_HHmmss"
$jsonPath = Join-Path $OutputDir "real_candidate_proof_$stamp.json"
$mdPath = Join-Path $OutputDir "real_candidate_proof_$stamp.md"
$report | ConvertTo-Json -Depth 12 | Set-Content $jsonPath

$lines = @("# Real Candidate Proof", "", "- Status: $status", "- Generated: $($report.generatedAt)", "- Candidates created: $($report.summary.candidatesCreated)", "- Blocked: $blockedCount", "- Skipped: $($report.summary.skippedCandidates)", "- Paper tickets created: $ticketCount", "", "## Candidates", "")
if (@($candidates).Count -eq 0) { $lines += "No candidates were created." }
foreach ($candidate in @($candidates)) {
  $lines += "### $($candidate.symbol) - $($candidate.candidate_id)"
  $lines += ""
  foreach ($field in @("input_source","catalyst_summary","entry","stop","target","evidence_status","gate_status","risk_status","approval_status","paper_ticket_status","candidate_status","reject_reasons","warning_reasons")) {
    $value = $candidate.$field
    if ($value -is [array]) { $value = $value -join "; " }
    $lines += "- ${field}: $value"
  }
  $lines += ""
}
$lines += @("## Safety", "", "- Execution instructions created: $($safety.executionInstructions)", "- Unsafe paper tickets: $($safety.unsafeTickets)", "- Leveraged candidates: $($safety.leveragedCandidates)")
$lines | Set-Content $mdPath

Write-Host ""
Write-Host "Proof result: $status" -ForegroundColor Cyan
Write-Host "Report: $mdPath"
Write-Host "JSON:   $jsonPath"
foreach ($candidate in @($candidates)) {
  Write-Host ("{0}: candidate={1}; evidence={2}; gate={3}; risk={4}; approval={5}; ticket={6}" -f $candidate.symbol, $candidate.candidate_status, $candidate.evidence_status, $candidate.gate_status, $candidate.risk_status, $candidate.approval_status, $candidate.paper_ticket_status)
  Write-Host ("  rejects: {0}" -f (@($candidate.reject_reasons) -join "; "))
  Write-Host ("  warnings: {0}" -f (@($candidate.warning_reasons) -join "; "))
}

if ($status -ne "candidate_produced") { exit 1 }
