param(
    [string]$Manifest = "config/ai-shadow-benchmark-manifest-v1.json",
    [int]$MaxEvents = 0,
    [switch]$NonInteractive,
    [string]$DatabaseUrl = ""
)

$ErrorActionPreference = "Stop"

function Set-DefaultEnvironmentValue {
    param([string]$Name, [string]$Value)
    if ([string]::IsNullOrWhiteSpace([Environment]::GetEnvironmentVariable($Name))) {
        [Environment]::SetEnvironmentVariable($Name, $Value, "Process")
    }
}

Set-DefaultEnvironmentValue "JAX_AI_SHADOW_ENABLED" "true"
Set-DefaultEnvironmentValue "JAX_AI_PROVIDER" "ollama"
Set-DefaultEnvironmentValue "JAX_AI_MODEL" "qwen3.5:9b"
Set-DefaultEnvironmentValue "JAX_AI_BASE_URL" "http://localhost:11434"
Set-DefaultEnvironmentValue "JAX_AI_TIMEOUT_SECONDS" "120"
Set-DefaultEnvironmentValue "JAX_AI_TEMPERATURE" "0"
Set-DefaultEnvironmentValue "JAX_AI_SEED" "20260803"
Set-DefaultEnvironmentValue "JAX_AI_MAX_EVENTS" "60"

Set-DefaultEnvironmentValue "JAX_RUNTIME_MODE" "paper"
Set-DefaultEnvironmentValue "ALLOW_LIVE_TRADING" "false"
Set-DefaultEnvironmentValue "EXECUTION_ENABLED" "false"
Set-DefaultEnvironmentValue "EXECUTION_INSTRUCTION_WORKER_ENABLED" "false"
Set-DefaultEnvironmentValue "BROKER_EXECUTION_ALLOWED" "false"
Set-DefaultEnvironmentValue "MAX_LEVERAGE" "1"

if ($MaxEvents -gt 0) {
    $env:JAX_AI_MAX_EVENTS = $MaxEvents.ToString()
}
if (-not [string]::IsNullOrWhiteSpace($DatabaseUrl)) {
    $env:DATABASE_URL = $DatabaseUrl
}
Set-DefaultEnvironmentValue "DATABASE_URL" "postgresql://jax:jax@localhost:5433/jax"

$selectedCount = [int]$env:JAX_AI_MAX_EVENTS
$timeoutSeconds = [int]$env:JAX_AI_TIMEOUT_SECONDS
$upperBoundMinutes = [math]::Ceiling(($selectedCount * $timeoutSeconds * 2) / 60)
$command = "go run ./cmd/ai-shadow-benchmark --manifest `"$Manifest`" --execute"

Write-Host "AI shadow benchmark preflight" -ForegroundColor Cyan
Write-Host "  Manifest: $Manifest"
Write-Host "  Planned event count: $selectedCount"
Write-Host "  Model: $env:JAX_AI_MODEL"
Write-Host "  Provider: $env:JAX_AI_PROVIDER"
Write-Host "  Ollama: $env:JAX_AI_BASE_URL"
Write-Host "  Temperature / seed: $env:JAX_AI_TEMPERATURE / $env:JAX_AI_SEED"
Write-Host "  Timeout per request: $timeoutSeconds seconds"
Write-Host "  Estimated worst-case runtime: up to $upperBoundMinutes minutes (includes one retry per event)"
Write-Host "  Runtime safety: paper; live/execution/worker/broker disabled; maximum leverage 1x"
Write-Host "  Command: $command"

& go run ./cmd/ai-shadow-benchmark --manifest $Manifest --preflight
if ($LASTEXITCODE -ne 0) {
    throw "AI shadow preflight failed. Ollama, model, database, migration, configuration, or manifest is unavailable."
}

if (-not $NonInteractive) {
    $confirmation = Read-Host "Type RUN to execute this real bounded model batch"
    if ($confirmation -cne "RUN") {
        Write-Host "Benchmark cancelled; no model requests were sent."
        exit 0
    }
} else {
    Write-Host "Non-interactive execution explicitly enabled by -NonInteractive." -ForegroundColor Yellow
}

& go run ./cmd/ai-shadow-benchmark --manifest $Manifest --execute
exit $LASTEXITCODE
