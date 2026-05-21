param(
  [ValidateSet("quick", "full")]
  [string]$Mode = "quick",
  [string]$ApiBase = "http://localhost:8081",
  [string]$ResearchBase = "http://localhost:8091",
  [string]$IbBridgeBase = "http://localhost:8092",
  [string]$Agent0Base = "http://localhost:8093",
  [string]$OutputDir = "Docs/runs",
  [switch]$OpenVisualReport
)

$ErrorActionPreference = "Stop"

$forwardedArgs = @(
  "-NoProfile",
  "-ExecutionPolicy", "Bypass",
  "-File", "scripts/test-platform.ps1",
  "-Mode", $Mode,
  "-ApiBase", $ApiBase,
  "-ResearchBase", $ResearchBase,
  "-IbBridgeBase", $IbBridgeBase,
  "-Agent0Base", $Agent0Base,
  "-OutputDir", $OutputDir
)

if ($OpenVisualReport.IsPresent) {
  $forwardedArgs += "-OpenVisualReport"
}

& powershell @forwardedArgs
