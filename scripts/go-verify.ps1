param(
  [ValidateSet("quick", "standard", "full")]
  [string]$Mode = "standard",
  [string[]]$Packages = @("./...")
)

$ErrorActionPreference = "Stop"

function Invoke-Step {
  param(
    [string]$Name,
    [scriptblock]$Action
  )
  Write-Host "== $Name =="
  & $Action
}

function Assert-GoAvailable {
  if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw "Go is not available on PATH."
  }
}

function Run-GoFmtCheck {
  $files = git ls-files "*.go" | Where-Object { $_ -notlike "archive/*" }
  if (-not $files) {
    return
  }

  $files = gofmt -l $files
  if ($files) {
    Write-Host "gofmt would modify:"
    $files | ForEach-Object { Write-Host "  $_" }
    throw "Formatting check failed. Run: gofmt -w <listed files>"
  }
}

function Run-GoTest {
  param([string[]]$Targets)
  go test @Targets
}

function Run-Lint {
  param([string[]]$Targets)
  $golangci = Get-Command golangci-lint -ErrorAction SilentlyContinue
  if ($null -eq $golangci) {
    Write-Host "golangci-lint not installed; skipping lint."
    return
  }
  golangci-lint run @Targets
}

function Get-ActivePackages {
  $all = go list ./...
  return $all | Where-Object { $_ -notmatch "/archive/" }
}

Assert-GoAvailable

switch ($Mode) {
  "quick" {
    Invoke-Step -Name "go test (quick)" -Action { Run-GoTest -Targets $Packages }
  }
  "standard" {
    Invoke-Step -Name "gofmt check" -Action { Run-GoFmtCheck }
    Invoke-Step -Name "go test (standard)" -Action { Run-GoTest -Targets $Packages }
  }
  "full" {
    $activePackages = Get-ActivePackages
    Invoke-Step -Name "gofmt check" -Action { Run-GoFmtCheck }
    Invoke-Step -Name "golangci-lint (full)" -Action { Run-Lint -Targets $activePackages }
    Invoke-Step -Name "go test (full)" -Action { Run-GoTest -Targets $activePackages }
  }
}
