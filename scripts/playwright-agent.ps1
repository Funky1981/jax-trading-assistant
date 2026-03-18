# scripts/playwright-agent.ps1
# Lightweight HTTP agent that runs Playwright E2E tests on the host machine
# on behalf of the Docker-based jax-trader backend.
#
# The jax-trader container cannot run `npx playwright test` because Node.js
# is not installed inside the image.  This agent listens on port 9092 and
# is reachable from the container via http://host.docker.internal:9092.
#
# Endpoints
#   POST /run?spec=<name>   Run tests (optionally scoped to one spec file).
#                           Blocks until tests complete, returns JSON result.
#   GET  /health            Returns {"status":"ok"}.
#
# Usage: automatically started by start.ps1; stopped by stop.ps1.

param(
    [int]$Port = 9092
)

$ErrorActionPreference = "Stop"

$RepoRoot   = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$FrontendDir = Join-Path $RepoRoot "frontend"
$RuntimeDir  = Join-Path $RepoRoot ".runtime"
$PidFile     = Join-Path $RuntimeDir "playwright-agent.pid"
$LogFile     = Join-Path (Join-Path $RepoRoot "logs") "playwright-agent.log"

# Prefer the locally-installed playwright CLI over a global one.
$PlaywrightCli = Join-Path $FrontendDir "node_modules\playwright\cli.js"
if (-not (Test-Path $PlaywrightCli)) {
    Write-Error "Playwright not found at $PlaywrightCli -- run 'npm install' in frontend/ first."
    exit 1
}

# Ensure runtime dir exists and write our PID.
if (-not (Test-Path $RuntimeDir)) { New-Item -ItemType Directory $RuntimeDir | Out-Null }
$PID | Set-Content $PidFile

function Write-Log([string]$msg) {
    $ts = (Get-Date -Format "yyyy-MM-dd HH:mm:ss")
    "$ts  $msg" | Tee-Object -FilePath $LogFile -Append | Out-Null
    Write-Host "$ts  $msg"
}

function Send-Json($ctx, [int]$code, $obj) {
    $json  = $obj | ConvertTo-Json -Depth 5 -Compress
    $bytes = [System.Text.Encoding]::UTF8.GetBytes($json)
    $ctx.Response.StatusCode      = $code
    $ctx.Response.ContentType     = "application/json; charset=utf-8"
    $ctx.Response.ContentLength64 = $bytes.Length
    $ctx.Response.OutputStream.Write($bytes, 0, $bytes.Length)
    $ctx.Response.OutputStream.Close()
}

# Docker Desktop for Windows routes host.docker.internal → host loopback,
# so listening on localhost is sufficient and requires no admin rights.
$listener = [System.Net.HttpListener]::new()
$listener.Prefixes.Add("http://localhost:$Port/")

try {
    $listener.Start()
} catch {
    Write-Error "Failed to start HTTP listener on port ${Port}: $_"
    exit 1
}

Write-Log "Playwright agent listening on http://localhost:$Port  (PID $PID)"

# Track the last result so GET /health can also include it.
$script:LastResult = $null

try {
    while ($listener.IsListening) {
        $ctx = $null
        try {
            $ctx = $listener.GetContext()
        } catch {
            # Listener was stopped.
            break
        }

        $req  = $ctx.Request
        $path = $req.Url.AbsolutePath
        $meth = $req.HttpMethod.ToUpper()

        if ($path -eq "/health" -and $meth -eq "GET") {
            Send-Json $ctx 200 @{ status = "ok" }
            continue
        }

        if ($path -eq "/run" -and $meth -eq "POST") {
            # Parse ?spec= query param.
            $qs   = [System.Web.HttpUtility]::ParseQueryString($req.Url.Query)
            $spec = $qs["spec"]

            # Build playwright args.  Sanitise spec to a bare filename.
            $args = @("test", "--reporter=list")
            if ($spec) {
                $cleaned = [System.IO.Path]::GetFileNameWithoutExtension($spec) `
                    -replace "\.\.", "" `
                    -replace "[/\\]", ""
                if ($cleaned) {
                    $args += "e2e/$cleaned.spec.ts"
                    Write-Log "Running spec: $cleaned"
                } else {
                    Write-Log "Skipped invalid spec '$spec', running full suite"
                }
            } else {
                Write-Log "Running full test suite"
            }

            $started   = Get-Date
            $outFile   = Join-Path $env:TEMP "pw-out-$([System.Diagnostics.Process]::GetCurrentProcess().Id).txt"
            $errFile   = Join-Path $env:TEMP "pw-err-$([System.Diagnostics.Process]::GetCurrentProcess().Id).txt"

            try {
                $proc = Start-Process `
                    -FilePath "node" `
                    -ArgumentList (@($PlaywrightCli) + $args) `
                    -WorkingDirectory $FrontendDir `
                    -RedirectStandardOutput $outFile `
                    -RedirectStandardError  $errFile `
                    -PassThru -Wait -NoNewWindow
                $exitCode = $proc.ExitCode
            } catch {
                Write-Log "Failed to run playwright: $_"
                $exitCode = -1
            }

            $completed = Get-Date
            $output    = ""
            if (Test-Path $outFile) { $output += (Get-Content $outFile -Raw -ErrorAction SilentlyContinue) }
            if (Test-Path $errFile) { $output += (Get-Content $errFile -Raw -ErrorAction SilentlyContinue) }
            Remove-Item $outFile, $errFile -Force -ErrorAction SilentlyContinue

            $status = if ($exitCode -eq 0) { "passed" } else { "failed" }
            Write-Log "Run finished: $status (exit $exitCode) in $([int]($completed - $started).TotalSeconds)s"

            # Truncate output to 24 KB to match the Go-side limit.
            $maxBytes = 24000
            if ($output.Length -gt $maxBytes) {
                $output = $output.Substring(0, $maxBytes) + "`n... [truncated]"
            }

            $result = @{
                startedAt   = $started.ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
                completedAt = $completed.ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
                durationMs  = [int64]($completed - $started).TotalMilliseconds
                exitCode    = $exitCode
                status      = $status
                spec        = if ($spec) { $spec } else { "" }
                output      = $output
            }
            $script:LastResult = $result
            Send-Json $ctx 200 $result
            continue
        }

        # 404 for anything else.
        $ctx.Response.StatusCode = 404
        $ctx.Response.OutputStream.Close()
    }
} finally {
    $listener.Stop()
    Remove-Item $PidFile -Force -ErrorAction SilentlyContinue
    Write-Log "Playwright agent stopped"
}
