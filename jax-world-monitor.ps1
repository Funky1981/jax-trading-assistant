param(
    [ValidateSet("up", "down", "status", "logs")]
    [string]$Action = "up"
)

$ErrorActionPreference = "Stop"
$repo = $PSScriptRoot
$compose = Join-Path $repo "docker-compose.yml"

switch ($Action) {
    "up" {
        docker compose -f $compose --project-directory $repo up -d --build
    }
    "down" {
        docker compose -f $compose --project-directory $repo down
    }
    "status" {
        docker compose -f $compose --project-directory $repo ps
    }
    "logs" {
        docker compose -f $compose --project-directory $repo logs --tail=150 worldmonitor-events jax-trader
    }
}

if ($LASTEXITCODE -ne 0) {
    throw "docker compose $Action failed with exit code $LASTEXITCODE"
}
