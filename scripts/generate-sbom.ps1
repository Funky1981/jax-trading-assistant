[CmdletBinding()]
param(
    [string]$OutputDirectory = (Join-Path (Get-Location) 'reports/sbom'),
    [switch]$IncludeImages
)

$ErrorActionPreference = 'Stop'

$SyftImage = 'anchore/syft:v1.51.1@sha256:95fe0835e5bebc6f8b1f8acef68d47d63d594ef4c0f25c097ff853b23cbac74c'
$Repository = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$OutputDirectory = [IO.Path]::GetFullPath($OutputDirectory)

if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
    throw 'Docker is required to generate the reproducible Syft SBOM set.'
}

New-Item -ItemType Directory -Force -Path $OutputDirectory | Out-Null
$GeneratedOutputFiles = [Collections.Generic.List[string]]::new()

function Convert-ToContainerPath([string]$Path) {
    return $Path.Replace('\', '/')
}

function Invoke-SyftFile([string]$RelativePath, [string]$OutputName, [string]$Cataloger, [string]$Enrichment) {
    $containerOutput = "/out/$OutputName"
    $arguments = @(
        'run', '--rm',
        '--mount', "type=bind,src=$Repository,dst=/src,readonly",
        '--mount', "type=bind,src=$OutputDirectory,dst=/out",
        $SyftImage, 'scan', "file:/src/$(Convert-ToContainerPath $RelativePath)",
        '--output', "cyclonedx-json=$containerOutput",
        '--override-default-catalogers', $Cataloger,
        '--enrich', $Enrichment
    )
    & docker @arguments
    if ($LASTEXITCODE -ne 0) {
        throw "Syft failed for $RelativePath with exit code $LASTEXITCODE."
    }
    $GeneratedOutputFiles.Add((Join-Path $OutputDirectory $OutputName))
}

$modulePaths = @('go.mod') + @(
    & rg --files -g 'go.mod' (Join-Path $Repository 'libs') (Join-Path $Repository 'tools') |
        Where-Object { $_ -notmatch '\\archive\\' }
) | ForEach-Object {
    if ([IO.Path]::IsPathRooted($_)) {
        $_.Substring($Repository.Length + 1)
    } else {
        $_
    }
} | Sort-Object -Unique

foreach ($modulePath in $modulePaths) {
    $relativeDirectory = Split-Path $modulePath -Parent
    $stem = if ([string]::IsNullOrEmpty($relativeDirectory)) { 'root' } else { $relativeDirectory.Replace('\', '-').Replace('/', '-') }
    Invoke-SyftFile $modulePath "go-$stem.cdx.json" 'go-module-file-cataloger' 'golang'
}

Invoke-SyftFile 'services/ib-bridge/requirements.txt' 'ib-bridge-python.cdx.json' 'python-package-cataloger' 'python'

$frontendOutput = Join-Path $OutputDirectory 'frontend-runtime.cdx.json'
Push-Location (Join-Path $Repository 'frontend')
try {
    & npm sbom --package-lock-only --omit=dev --sbom-format cyclonedx | Out-File -FilePath $frontendOutput -Encoding utf8
    if ($LASTEXITCODE -ne 0) {
        throw "npm sbom failed with exit code $LASTEXITCODE."
    }
    $GeneratedOutputFiles.Add($frontendOutput)
}
finally {
    Pop-Location
}

if ($IncludeImages) {
    $imageOutput = Join-Path $OutputDirectory 'images'
    New-Item -ItemType Directory -Force -Path $imageOutput | Out-Null
    $images = @(
        'pgvector/pgvector:pg16@sha256:ccc6e83d6e35e931dc7c5def2022729d5a6c370318d099181995567ff1fb4d6b',
        'prom/prometheus@sha256:5ce7540c3c00ef4ab0c9d2c995c6a5b9c421f44b4a115d97a2c7af3b1c21cbb0',
        'grafana/grafana@sha256:f772d434e8fab0049deb2b1b30abd43342bcfca1537614aa8d36080232cf4283',
        'postgres:16-alpine@sha256:cf78e76683b9ca8c5733cbbdce6c9262b45b6767934dd0a95e671f9a0fc20685',
        'postgres:16@sha256:f1c3376c26f2609ab9f29f71f824103fe2fcd8ee0346485cb6122a4f93df6f94',
        'qdrant/qdrant@sha256:12364fe851b9f17356fc88189fc06d1b521262e04659ec7345975b00c9246a10',
        'python:3.11-slim@sha256:9534e5a8e315485d4061ed659af0fd78a284c015f9b73661b41d6bab25604534',
        'golang:1.25.4-alpine@sha256:d3f0cf7723f3429e3f9ed846243970b20a2de7bae6a5b66fc5914e228d831bbb',
        'alpine:3.19@sha256:6baf43584bcb78f2e5847d1de515f23499913ac9f12bdf834811a3145eb11ca1',
        'node:22-alpine@sha256:c610fcdfb1d5b4740dd70c284ed3cb16bb857e0f7166196e36a5501df7a3aa32',
        'nginx:1.27-alpine@sha256:65645c7bb6a0661892a8b03b89d0743208a18dd2f3f17a54ef4b76fb8e2f2a10'
    )
    foreach ($image in $images) {
        $safeName = ($image -replace '[^A-Za-z0-9.-]', '_') + '.cdx.json'
        & docker run --rm --mount "type=bind,src=$imageOutput,dst=/out" $SyftImage scan "registry:$image" --output "cyclonedx-json=/out/$safeName" | Out-Null
        if ($LASTEXITCODE -ne 0) {
            throw "Syft registry scan failed for $image with exit code $LASTEXITCODE."
        }
        $GeneratedOutputFiles.Add((Join-Path $imageOutput $safeName))
    }
}

foreach ($generatedFile in $GeneratedOutputFiles) {
    $bom = Get-Content -LiteralPath $generatedFile -Raw | ConvertFrom-Json
    if ($bom.bomFormat -ne 'CycloneDX') {
        throw "Unexpected SBOM format in $([IO.Path]::GetFileName($generatedFile))."
    }
}

Write-Output "SBOM set generated under $OutputDirectory using pinned Syft image $SyftImage."
