#!/usr/bin/env pwsh

<#
.SYNOPSIS
    Database migration management script

.DESCRIPTION
    Manages Postgres database migrations for Jax Trading Assistant

.PARAMETER Action
    The migration action to perform: up, down, version, create, force

.PARAMETER Steps
    Number of steps for up/down migrations (default: all for up, 1 for down)

.PARAMETER Name
    Name for new migration (used with 'create' action)

.PARAMETER Version
    Version to force migration to (used with 'force' action)

.EXAMPLE
    .\migrate.ps1 up
    Applies all pending migrations

.EXAMPLE
    .\migrate.ps1 down -Steps 1
    Rolls back the last migration

.EXAMPLE
    .\migrate.ps1 create -Name "add_user_table"
    Creates a new migration file pair

.EXAMPLE
    .\migrate.ps1 version
    Shows current migration version
#>

param(
    [Parameter(Mandatory=$true, Position=0)]
    [ValidateSet('up', 'down', 'version', 'create', 'force')]
    [string]$Action,

    [Parameter(Mandatory=$false)]
    [int]$Steps,

    [Parameter(Mandatory=$false)]
    [string]$Name,

    [Parameter(Mandatory=$false)]
    [int]$Version
)

# Configuration
$MigrationsPath = "db/postgres/migrations"
$DatabaseURL = $env:DATABASE_URL
if (-not $DatabaseURL) {
    $DatabaseURL = "postgres://jaxuser:jaxpass@localhost:5432/jaxdb?sslmode=disable"
    Write-Host "Using default DATABASE_URL: $DatabaseURL" -ForegroundColor Yellow
}

# Check if migrate CLI is installed
$migratePath = Get-Command migrate -ErrorAction SilentlyContinue
if (-not $migratePath) {
    Write-Host "Error: migrate CLI not found. Install with:" -ForegroundColor Red
    Write-Host "  go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest" -ForegroundColor Yellow
    exit 1
}

# Execute migration action
switch ($Action) {
    'up' {
        if ($Steps) {
            Write-Host "Running $Steps migration(s) up..." -ForegroundColor Cyan
            migrate -path $MigrationsPath -database $DatabaseURL up $Steps
        } else {
            Write-Host "Running all migrations up..." -ForegroundColor Cyan
            migrate -path $MigrationsPath -database $DatabaseURL up
        }
    }
    'down' {
        $downSteps = if ($Steps) { $Steps } else { 1 }
        Write-Host "Rolling back $downSteps migration(s)..." -ForegroundColor Cyan
        migrate -path $MigrationsPath -database $DatabaseURL down $downSteps
    }
    'version' {
        Write-Host "Checking migration version..." -ForegroundColor Cyan
        migrate -path $MigrationsPath -database $DatabaseURL version
    }
    'create' {
        if (-not $Name) {
            Write-Host "Error: -Name parameter required for create action" -ForegroundColor Red
            exit 1
        }
        Write-Host "Creating migration: $Name" -ForegroundColor Cyan
        migrate create -ext sql -dir $MigrationsPath -seq $Name
    }
    'force' {
        if (-not $Version) {
            Write-Host "Error: -Version parameter required for force action" -ForegroundColor Red
            exit 1
        }
        Write-Host "Forcing migration version to: $Version" -ForegroundColor Yellow
        migrate -path $MigrationsPath -database $DatabaseURL force $Version
    }
}

if ($LASTEXITCODE -eq 0) {
    Write-Host "✓ Migration action completed successfully" -ForegroundColor Green
} else {
    Write-Host "✗ Migration action failed with exit code: $LASTEXITCODE" -ForegroundColor Red
    exit $LASTEXITCODE
}
