#!/usr/bin/env pwsh
# scripts/smoke-memory.ps1
#
# Smoke test for the memory API and end-to-end retain/recall cycle.
#
# Prerequisites:
#   - Postgres with pgvector running (docker compose up postgres, or external)
#   - golang-migrate CLI installed (https://github.com/golang-migrate/migrate)
#   - EMBEDDING_PROVIDER=local (default) for zero-cost local embeddings
#   - or EMBEDDING_PROVIDER=openai with OPENAI_API_KEY set
#
# Usage:
#   .\scripts\smoke-memory.ps1
#   .\scripts\smoke-memory.ps1 -DatabaseURL "postgresql://jax:jax@localhost:5433/jax?sslmode=disable"
#   .\scripts\smoke-memory.ps1 -SkipEmbed

param(
    [string]$DatabaseURL = "",
    [string]$ResearchURL = "",
    [ValidateSet("local", "openai")]
    [string]$EmbeddingProvider = "",
    [switch]$SkipEmbed
)

if (-not $DatabaseURL) { $DatabaseURL = if ($env:TEST_DATABASE_URL) { $env:TEST_DATABASE_URL } else { "postgresql://jax:jax@localhost:5433/jax?sslmode=disable" } }
if (-not $ResearchURL)  { $ResearchURL  = if ($env:RESEARCH_URL)       { $env:RESEARCH_URL }       else { "http://localhost:8091" } }
if (-not $EmbeddingProvider) {
    $EmbeddingProvider = if ($env:EMBEDDING_PROVIDER) { $env:EMBEDDING_PROVIDER.ToLowerInvariant() } else { "local" }
}
$RequiredSchemaVersion = if ($env:MEMORY_REQUIRED_SCHEMA_VERSION) { [int]$env:MEMORY_REQUIRED_SCHEMA_VERSION } else { 21 }

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
if ($PSVersionTable.PSVersion.Major -ge 7) {
    $PSNativeCommandUseErrorActionPreference = $true
}

$Green  = "`e[32m"
$Red    = "`e[31m"
$Yellow = "`e[33m"
$Reset  = "`e[0m"

function Write-Step([string]$msg) { Write-Host "${Yellow}>> $msg${Reset}" }
function Write-OK([string]$msg)   { Write-Host "${Green}PASS: $msg${Reset}" }
function Write-Fail([string]$msg) { Write-Host "${Red}FAIL: $msg${Reset}"; exit 1 }

# 1. Run migrations ----------------------------------------------------------
Write-Step "Applying migrations to $DatabaseURL"

if (Get-Command docker -ErrorAction SilentlyContinue) {
    $migrationsPath = Join-Path (Join-Path (Join-Path (Join-Path $PSScriptRoot "..") "db") "postgres") "migrations"
    $migrationsPath = (Resolve-Path $migrationsPath).Path
    $oldPref = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    $result = docker run --rm --network host `
        -v "${migrationsPath}:/migrations" `
        migrate/migrate `
        -path=/migrations -database $DatabaseURL up 2>&1
    $exitCode = $LASTEXITCODE
    $ErrorActionPreference = $oldPref
    $resultText = ($result | ForEach-Object { $_.ToString() }) -join ' '
    Write-Host $resultText
    if ($exitCode -ne 0 -and $resultText -notmatch "no change") {
        Write-Fail "Migration failed (exit $exitCode)"
    }
    Write-OK "Migrations applied"
} else {
    Write-Host "${Yellow}  docker not found -- skipping automatic migration.${Reset}"
    Write-Host "  Run manually: docker run --rm --network host -v `"\${PWD}/db/postgres/migrations:/migrations`" migrate/migrate -path=/migrations -database '$DatabaseURL' up"
}

# 2. Verify migration state and memory schema --------------------------------
Write-Step "Checking migration state and memory schema"

if (Get-Command psql -ErrorAction SilentlyContinue) {
    $migrationVersion = psql $DatabaseURL -t -c "SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1;" 2>&1
    if ($migrationVersion -notmatch "\b$RequiredSchemaVersion\b") {
        Write-Fail "expected schema_migrations version $RequiredSchemaVersion, got: $migrationVersion"
    }
    Write-OK "schema_migrations version = $RequiredSchemaVersion"

    $tableCheck = psql $DatabaseURL -t -c "SELECT to_regclass('public.memory_items');" 2>&1
    if ($tableCheck -notmatch "memory_items") {
        Write-Fail "memory_items table not found -- run migration first"
    }
    Write-OK "memory_items table exists"

    # Check vector extension
    $extCheck = psql $DatabaseURL -t -c "SELECT extname FROM pg_extension WHERE extname='vector';" 2>&1
    if ($extCheck -notmatch "vector") {
        Write-Fail "pgvector extension not installed in Postgres"
    }
    Write-OK "pgvector extension active"
} else {
    Write-Host "${Yellow}  psql not found -- skipping table check.${Reset}"
}

# 3. Health check against jax-research -------------------------------------
Write-Step "Checking jax-research readiness at $ResearchURL"

try {
    $health = Invoke-RestMethod "$ResearchURL/ready" -TimeoutSec 5
    if ($health.status -ne "healthy") {
        Write-Fail "jax-research health = $($health.status)"
    }
    if (-not $health.memory_ready) {
        Write-Fail "jax-research memory_ready = false ($($health.memory_error))"
    }
    if ($health.embedding_provider -and $health.embedding_provider -ne $EmbeddingProvider) {
        Write-Fail "jax-research embedding_provider = $($health.embedding_provider), expected $EmbeddingProvider"
    }
    Write-OK "jax-research ready (v$($health.version))"
} catch {
    Write-Host "${Yellow}  jax-research not reachable -- skipping HTTP smoke tests.${Reset}"
    Write-Host "  Start with: docker compose up jax-research"
    exit 0
}

if ($EmbeddingProvider -eq "openai" -and -not $env:OPENAI_API_KEY) {
    Write-Fail "openai smoke requested but OPENAI_API_KEY is not set"
}

# 4. List banks -------------------------------------------------------------
Write-Step "GET /v1/memory/banks"
$banks = Invoke-RestMethod "$ResearchURL/v1/memory/banks" -TimeoutSec 10
$requiredBanks = @("research","trades","signals","reflections")
foreach ($b in $requiredBanks) {
    if ($banks -notcontains $b) { Write-Fail "bank '$b' missing from /v1/memory/banks" }
}
Write-OK "banks = $($banks -join ', ')"

# 5. Retain a memory item ---------------------------------------------------
Write-Step "POST /tools  [memory.retain]"

$retainPayload = @{
    tool = "memory.retain"
    input = @{
        bank = "research"
        item = @{
            ts      = (Get-Date -Format "o")
            type    = "signal"
            symbol  = "AAPL"
            summary = "Smoke test: MACD crossover on AAPL daily $(Get-Date -Format 'yyyyMMddHHmmss')"
            tags    = @("smoke", "aapl")
            data    = @{ confidence = 0.77 }
            source  = @{ system = "smoke-test" }
        }
    }
} | ConvertTo-Json -Depth 10

try {
    $retainResp = Invoke-RestMethod "$ResearchURL/tools" -Method Post `
        -Body $retainPayload -ContentType "application/json" -TimeoutSec 30
    $retainedID = $retainResp.output.id
    if (-not $retainedID) { Write-Fail "retain response missing id: $($retainResp | ConvertTo-Json)" }
    Write-OK "retained id=$retainedID"
} catch {
    if ($EmbeddingProvider -eq "openai" -and -not $env:OPENAI_API_KEY) {
        Write-Fail "retain failed in openai mode because OPENAI_API_KEY is not set"
    }
    if ($SkipEmbed) {
        Write-Host "${Yellow}  retain failed -- skipped because -SkipEmbed was passed.${Reset}"
        exit 0
    }
    Write-Fail "retain failed: $_"
}

# 6. Recall (structured) ----------------------------------------------------
Write-Step "POST /tools  [memory.recall structured]"
$recallPayload = @{
    tool = "memory.recall"
    input = @{
        bank  = "research"
        query = @{ symbol = "AAPL"; limit = 5 }
    }
} | ConvertTo-Json -Depth 10

$recallResp = Invoke-RestMethod "$ResearchURL/tools" -Method Post `
    -Body $recallPayload -ContentType "application/json" -TimeoutSec 15
if ($null -eq $recallResp.output.items) { Write-Fail "recall response missing items" }
Write-OK "recall returned $($recallResp.output.items.Count) item(s)"

# 7. Recall (vector) --------------------------------------------------------
Write-Step "POST /tools  [memory.recall vector]"
$vecPayload = @{
    tool = "memory.recall"
    input = @{
        bank  = "research"
        query = @{ q = "MACD crossover AAPL signal"; limit = 5 }
    }
} | ConvertTo-Json -Depth 10

$vecResp = Invoke-RestMethod "$ResearchURL/tools" -Method Post `
    -Body $vecPayload -ContentType "application/json" -TimeoutSec 30
if ($null -eq $vecResp.output.items) { Write-Fail "vector recall response missing items" }
Write-OK "vector recall returned $($vecResp.output.items.Count) item(s)"

# 8. GET /v1/memory/banks/{bank}/items -------------------------------------
Write-Step "GET /v1/memory/banks/research/items"
$listResp = Invoke-RestMethod "$ResearchURL/v1/memory/banks/research/items" -TimeoutSec 10
if ($null -eq $listResp.items) { Write-Fail "items list response missing 'items' key" }
Write-OK "items list returned $($listResp.items.Count) item(s)"

# 9. GET /v1/memory/banks/{bank}/items/{id} --------------------------------
Write-Step "GET /v1/memory/banks/research/items/$retainedID"
$getResp = Invoke-RestMethod "$ResearchURL/v1/memory/banks/research/items/$retainedID" -TimeoutSec 10
if ($getResp.id -ne $retainedID) { Write-Fail "get-by-id returned id=$($getResp.id), want $retainedID" }
Write-OK "get-by-id OK  summary=$($getResp.summary.Substring(0,[Math]::Min(50,$getResp.summary.Length)))..."

# 10. Reflect ---------------------------------------------------------------
Write-Step "POST /tools  [memory.reflect]"
$reflectPayload = @{
    tool = "memory.reflect"
    input = @{
        bank   = "research"
        params = @{ query = "AAPL trading patterns from last week" }
    }
} | ConvertTo-Json -Depth 10

$reflectResp = Invoke-RestMethod "$ResearchURL/tools" -Method Post `
    -Body $reflectPayload -ContentType "application/json" -TimeoutSec 30
if ($null -eq $reflectResp.output.items -or $reflectResp.output.items.Count -eq 0) {
    Write-Fail "reflect returned no items"
}
Write-OK "reflect OK  summary=$($reflectResp.output.items[0].summary.Substring(0,[Math]::Min(80,$reflectResp.output.items[0].summary.Length)))"

# 11. /v1/memory/search -----------------------------------------------------
Write-Step "GET /v1/memory/search?q=MACD&bank=research"
$searchResp = Invoke-RestMethod "$ResearchURL/v1/memory/search?q=MACD&bank=research" -TimeoutSec 30
if ($null -eq $searchResp.items) { Write-Fail "search response missing 'items' key" }
Write-OK "search returned $($searchResp.items.Count) item(s)"

Write-Host ""
Write-OK "All memory smoke tests passed."
