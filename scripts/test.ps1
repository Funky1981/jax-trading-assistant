$ErrorActionPreference = "Stop"

Push-Location (Split-Path -Parent $PSScriptRoot)
try {
  if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw "Go not found on PATH. Install Go 1.22+ to run backend tests."
  }

  Write-Host "== gofmt check =="
  $files = gofmt -l .
  if ($files) {
    Write-Host "gofmt would modify:"
    $files | ForEach-Object { Write-Host "  $_" }
    throw "Run: gofmt -w ."
  }

  Write-Host "== golangci-lint =="
  $golangci = Get-Command golangci-lint -ErrorAction SilentlyContinue
  if ($golangci) {
    golangci-lint run ./...
  } else {
    Write-Host "golangci-lint not installed; skipping."
    Write-Host "Install (Go): go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
  }

  Write-Host "== go test =="
  go test ./...
}
finally {
  Pop-Location
}
