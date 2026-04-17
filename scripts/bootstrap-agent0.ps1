#!/usr/bin/env pwsh

param(
    [string]$Python = "python",
    [string]$ServiceDir = "services/agent0-service",
    [switch]$SkipInstall,
    [switch]$SkipTests
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
if ($PSVersionTable.PSVersion.Major -ge 7) {
    $PSNativeCommandUseErrorActionPreference = $true
}

function Resolve-VenvPython([string]$VenvPath) {
    $windowsPython = Join-Path $VenvPath "Scripts/python.exe"
    if (Test-Path $windowsPython) {
        return $windowsPython
    }

    $posixPython = Join-Path $VenvPath "bin/python"
    if (Test-Path $posixPython) {
        return $posixPython
    }

    throw "Unable to locate virtualenv python under $VenvPath"
}

$servicePath = (Resolve-Path $ServiceDir).Path
$venvPath = Join-Path $servicePath ".venv"

if (-not (Test-Path $venvPath)) {
    Write-Host "Creating Agent0 virtualenv at $venvPath"
    & $Python -m venv $venvPath
}

$venvPython = Resolve-VenvPython $venvPath

if (-not $SkipInstall) {
    Write-Host "Installing Agent0 dev requirements"
    & $venvPython -m pip install --upgrade pip
    & $venvPython -m pip install -r (Join-Path $servicePath "requirements-dev.txt")
}

Write-Host "Compiling Agent0 sources"
& $venvPython -m py_compile `
    (Join-Path $servicePath "agent.py") `
    (Join-Path $servicePath "config.py") `
    (Join-Path $servicePath "models.py") `
    (Join-Path $servicePath "main.py") `
    (Join-Path $servicePath "test_agent.py")

if (-not $SkipTests) {
    Write-Host "Running Agent0 unit tests"
    Push-Location $servicePath
    try {
        & $venvPython -m unittest test_agent.py -v
    }
    finally {
        Pop-Location
    }
}
