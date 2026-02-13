param(
  [ValidateSet("verify", "capture")]
  [string]$Mode = "verify"
)

$ErrorActionPreference = "Stop"

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
  throw "Go is not available on PATH."
}

if ($Mode -eq "capture") {
  Write-Host "== capture golden snapshots =="
  go run ./tests/golden/capture.go
}

Write-Host "== golden tests =="
go test -v ./tests/golden/... -tags=golden

Write-Host "== replay tests =="
go test ./tests/replay/...
