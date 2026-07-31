[CmdletBinding()]
param(
    [string]$DatabaseURL = "",
    [string]$OutputDirectory = ".runtime/evidence-quality",
    [string]$RulesetFile = "config/historical-evidence-quality-v1.json",
    [string]$Ruleset = ""
)

$ErrorActionPreference = "Stop"
$repo = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path

function Read-DotEnv {
    param([string]$Path)
    $values = @{}
    foreach ($line in Get-Content -LiteralPath $Path) {
        if ($line -match '^\s*#' -or $line -notmatch '=') { continue }
        $parts = $line.Split('=', 2)
        $values[$parts[0].Trim()] = $parts[1].Trim().Trim('"').Trim("'")
    }
    return $values
}

$dotenv = Read-DotEnv (Join-Path $repo ".env")
if ([string]::IsNullOrWhiteSpace($DatabaseURL)) { $DatabaseURL = $dotenv["DATABASE_URL"] }
if ([string]::IsNullOrWhiteSpace($DatabaseURL)) { throw "DATABASE_URL is required" }

if ($DatabaseURL -match '@postgres:5432') {
    $mapping = docker compose -f (Join-Path $repo "docker-compose.yml") port postgres 5432
    if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($mapping)) {
        throw "The Jax postgres service must be running before this read-only evaluation"
    }
    $port = ($mapping.Trim() -split ':')[-1]
    $DatabaseURL = $DatabaseURL -replace '@postgres:5432', "@127.0.0.1:$port"
}

$env:DATABASE_URL = $DatabaseURL
$env:JAX_RUNTIME_MODE = if ($env:JAX_RUNTIME_MODE) { $env:JAX_RUNTIME_MODE } elseif ($dotenv["JAX_TRADER_RUNTIME_MODE"]) { $dotenv["JAX_TRADER_RUNTIME_MODE"] } else { "paper" }
$env:ALLOW_LIVE_TRADING = if ($env:ALLOW_LIVE_TRADING) { $env:ALLOW_LIVE_TRADING } elseif ($dotenv["ALLOW_LIVE_TRADING"]) { $dotenv["ALLOW_LIVE_TRADING"] } else { "false" }
$env:EXECUTION_ENABLED = if ($env:EXECUTION_ENABLED) { $env:EXECUTION_ENABLED } elseif ($dotenv["EXECUTION_ENABLED"]) { $dotenv["EXECUTION_ENABLED"] } else { "false" }
$env:EXECUTION_INSTRUCTION_WORKER_ENABLED = if ($env:EXECUTION_INSTRUCTION_WORKER_ENABLED) { $env:EXECUTION_INSTRUCTION_WORKER_ENABLED } elseif ($dotenv["EXECUTION_INSTRUCTION_WORKER_ENABLED"]) { $dotenv["EXECUTION_INSTRUCTION_WORKER_ENABLED"] } else { "false" }
$env:BROKER_EXECUTION_ALLOWED = if ($env:BROKER_EXECUTION_ALLOWED) { $env:BROKER_EXECUTION_ALLOWED } elseif ($dotenv["BROKER_EXECUTION_ALLOWED"]) { $dotenv["BROKER_EXECUTION_ALLOWED"] } else { "false" }
$env:MAX_LEVERAGE = if ($env:MAX_LEVERAGE) { $env:MAX_LEVERAGE } elseif ($dotenv["MAX_LEVERAGE"]) { $dotenv["MAX_LEVERAGE"] } else { "1" }

Push-Location $repo
try {
    if ([string]::IsNullOrWhiteSpace($Ruleset)) {
        $Ruleset = (Get-Content -Raw -LiteralPath $RulesetFile | ConvertFrom-Json).version
    }
    go run ./cmd/evidence-quality-evaluation --ruleset $Ruleset --ruleset-file $RulesetFile --output-dir $OutputDirectory
    if ($LASTEXITCODE -ne 0) { throw "evidence quality evaluation failed" }
}
finally {
    Pop-Location
}
