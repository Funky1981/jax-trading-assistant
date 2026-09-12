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
        'python:3.11-slim@sha256:9534e5a8e315485d4061ed659af0fd78a284c015f9b73661b41d6bab25604534',
        'golang:1.26.8-alpine@sha256:ce864e7223ac17b1775e6fd0b4c0db580c2eb50e7953a427916379e4b92a1628',
        'alpine:3.23@sha256:fd791d74b68913cbb027c6546007b3f0d3bc45125f797758156952bc2d6daf40',
        'node:22-alpine@sha256:c610fcdfb1d5b4740dd70c284ed3cb16bb857e0f7166196e36a5501df7a3aa32',
        'nginx:1.29-alpine@sha256:5616878291a2eed594aee8db4dade5878cf7edcb475e59193904b198d9b830de'
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
