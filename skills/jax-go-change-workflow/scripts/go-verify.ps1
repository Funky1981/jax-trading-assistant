param(
  [ValidateSet("quick", "standard", "full")]
  [string]$Mode = "standard",

  [string[]]$Packages = @("./...")
)

$ErrorActionPreference = "Stop"

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
  throw "Go is not available on PATH."
}

function Run-GoTest {
  param([string[]]$Targets)
  if ($Targets.Count -eq 0) {
    $Targets = @("./...")
  }
  Write-Host "== go test $($Targets -join ' ') =="
  go test @Targets
}

if ($Mode -ne "quick") {
  Write-Host "== gofmt check =="
  $files = gofmt -l .
  if ($files) {
    Write-Host "gofmt would modify:"
    $files | ForEach-Object { Write-Host "  $_" }
    throw "Run gofmt -w . before verification."
  }
}

if ($Mode -eq "standard" -or $Mode -eq "full") {
  $lint = Get-Command golangci-lint -ErrorAction SilentlyContinue
  if ($lint) {
    Write-Host "== golangci-lint =="
    golangci-lint run ./...
  } else {
    Write-Host "golangci-lint not installed; skipping lint."
  }
}

Run-GoTest -Targets $Packages

if ($Mode -eq "full") {
  Write-Host "== go test -race ./... =="
  go test -race ./...
}
