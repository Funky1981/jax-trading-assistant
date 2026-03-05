param(
  [ValidateSet("all", "up", "ingest", "down")]
  [string]$Mode = "all",
  [switch]$DryRun,
  [string]$KnowledgeRoot = "knowledge/md",
  [string]$Dsn = "postgres://jax:jax@localhost:5433/jax?sslmode=disable",
  [string]$SchemaPath = "tools/sql/schema.sql"
)

$ErrorActionPreference = "Stop"

function Assert-Command {
  param([string]$Name)
  if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
    throw "$Name is not available on PATH."
  }
}

function Start-Db {
  Write-Host "== docker compose up postgres =="
  docker compose up -d postgres
}

function Apply-Schema {
  if (-not (Test-Path $SchemaPath)) {
    throw "Schema file not found: $SchemaPath"
  }
  Write-Host "== apply knowledge schema =="
  Get-Content $SchemaPath -Raw | docker compose exec -T postgres psql -U jax -d jax
}

function Run-Ingest {
  param([bool]$DryRunValue)
  if (-not (Test-Path $KnowledgeRoot)) {
    throw "Knowledge root not found: $KnowledgeRoot"
  }
  Write-Host "== ingest knowledge (dry-run=$DryRunValue) =="
  go run ./tools/cmd/ingest/main.go -root $KnowledgeRoot -dsn $Dsn -dry-run=$DryRunValue
}

function Stop-Db {
  Write-Host "== docker compose stop postgres =="
  docker compose stop postgres
}

Assert-Command -Name "docker"
Assert-Command -Name "go"

switch ($Mode) {
  "up" {
    Start-Db
    Apply-Schema
  }
  "ingest" {
    Run-Ingest -DryRunValue $DryRun.IsPresent
  }
  "down" {
    Stop-Db
  }
  "all" {
    Start-Db
    Apply-Schema
    Run-Ingest -DryRunValue $true
    if (-not $DryRun.IsPresent) {
      Run-Ingest -DryRunValue $false
    } else {
      Write-Host "Skipping full ingest because -DryRun was provided."
    }
  }
}
