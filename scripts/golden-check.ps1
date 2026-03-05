param(
  [ValidateSet("verify", "capture")]
  [string]$Mode = "verify",
  [string]$OutputDir = "tests/golden",
  [string]$BaseURL = "http://localhost"
)

$ErrorActionPreference = "Stop"

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
  throw "Go is not available on PATH."
}

switch ($Mode) {
  "capture" {
    Write-Host "== capture golden snapshots =="
    go run ./tests/golden/cmd/capture.go -output $OutputDir -base-url $BaseURL
  }
  "verify" {
    Write-Host "== compare utility tests =="
    go test ./tests/golden -count=1

    Write-Host "== golden tagged tests =="
    go test -tags=golden ./tests/golden/... -count=1
  }
}
