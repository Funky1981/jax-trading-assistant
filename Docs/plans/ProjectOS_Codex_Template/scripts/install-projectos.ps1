param(
    [Parameter(Mandatory=$true)]
    [string]$TargetRepoPath
)

$source = Split-Path -Parent $PSScriptRoot
$target = Resolve-Path $TargetRepoPath

Write-Host "Installing ProjectOS into $target"

$folders = @("project", "ai", ".github", "github-project-board")

foreach ($folder in $folders) {
    $sourcePath = Join-Path $source $folder
    $targetPath = Join-Path $target $folder

    if (Test-Path $sourcePath) {
        Copy-Item $sourcePath $targetPath -Recurse -Force
        Write-Host "Copied $folder"
    }
}

Write-Host "ProjectOS installed."
Write-Host "Next: complete /project/brief.md and run /ai/commands/baseline-existing-project.md"
