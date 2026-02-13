param(
  [ValidateSet("up", "schema", "ingest", "down", "all")]
  [string]$Mode = "all",

  [switch]$DryRun,

  [string]$DSN = "postgres://postgres:postgres@localhost:5432/jax_knowledge?sslmode=disable"
)

$ErrorActionPreference = "Stop"

function Run-Step {
  param([string]$Step)

  switch ($Step) {
    "up" {
      docker compose -f tools/docker-compose.yml up -d postgres
      return
    }
    "schema" {
      Get-Content tools/sql/schema.sql | docker compose -f tools/docker-compose.yml exec -T postgres psql -U postgres -d jax_knowledge -f /dev/stdin
      return
    }
    "ingest" {
      $dry = "false"
      if ($DryRun) {
        $dry = "true"
      }
      Push-Location tools
      try {
        go run ./cmd/ingest --root ../knowledge/md --dsn "$DSN" --dry-run=$dry
      } finally {
        Pop-Location
      }
      return
    }
    "down" {
      docker compose -f tools/docker-compose.yml down
      return
    }
    default {
      throw "Unknown step: $Step"
    }
  }
}

if ($Mode -eq "all") {
  Run-Step "up"
  Run-Step "schema"
  Run-Step "ingest"
  return
}

Run-Step $Mode
