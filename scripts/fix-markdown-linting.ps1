# Fix Markdown Linting Errors
# This script fixes common markdown linting errors across all .md files

param(
    [switch]$DryRun = $false
)

$ErrorActionPreference = 'Continue'
$filesFixed = 0
$errorsFixed = 0

# Get all markdown files
$mdFiles = Get-ChildItem -Path "c:\Projects\jax-trading assistant" -Recurse -Filter "*.md" | Where-Object {
    $_.FullName -notmatch 'node_modules' -and
    $_.FullName -notmatch '\\dist\\' -and
    $_.FullName -notmatch '\\.git\\'
}

Write-Host "Found $($mdFiles.Count) markdown files to process" -ForegroundColor Cyan

function Fix-MarkdownFile {
    param([string]$FilePath)
    
    $content = Get-Content -Path $FilePath -Raw -Encoding UTF8
    $original = $content
    $fixed = 0
    
    # Fix: Remove ````markdown wrapper at start/end of files
    if ($content -match '^````markdown\r?\n') {
        $content = $content -replace '^````markdown\r?\n', ''
        $content = $content -replace '\r?\n````\s*$', ''
        $fixed++
        Write-Host "  Fixed: Removed markdown wrapper" -ForegroundColor Green
    }
    
    # Fix MD040: Add language to fenced code blocks without language
    $content = $content -replace '(?m)^```\r?\n(?!``)', "``````text`n"
    if ($content -ne $original) {
        $matches = ([regex]::Matches($original, '(?m)^```\r?\n(?!``)')).Count
        if ($matches -gt 0) {
            $fixed += $matches
            Write-Host "  Fixed: Added 'text' to $matches code blocks" -ForegroundColor Green
        }
    }
    
    # Fix MD031: Blank lines around code blocks
    # Add blank line before code fence if missing
    $beforeFix = $content
    $content = $content -replace '(?m)([^\r\n])\r?\n(```)([a-z]*)\r?\n', "`$1`n`n``````$3`n"
    # Add blank line after code fence if missing  
    $content = $content -replace '(?m)\r?\n```\r?\n([^\r\n#])', "`n``````n`n`$1"
    
    # Fix MD022: Blank lines around headings
    # Add blank line before heading
    $content = $content -replace '(?m)([^\r\n])\r?\n(#{1,6} )', "`$1`n`n`$2"
    # Add blank line after heading
    $content = $content -replace '(?m)(#{1,6} .+)\r?\n([^\r\n#`*-])', "`$1`n`n`$2"
    
    # Fix MD009: Remove trailing spaces
    $beforeTrailing = $content
    $content = $content -replace '(?m)[ \t]+$', ''
    if ($content -ne $beforeTrailing) {
        $fixed++
        Write-Host "  Fixed: Removed trailing spaces" -ForegroundColor Green
    }
    
    # Fix MD010: Replace hard tabs with spaces
    if ($content -match '\t') {
        $tabCount = ([regex]::Matches($content, '\t')).Count
        $content = $content -replace '\t', '    '
        $fixed += $tabCount
        Write-Host "  Fixed: Replaced $tabCount tabs with spaces" -ForegroundColor Green
    }
    
    # Fix MD034: Wrap bare URLs (excluding already linked URLs)
    # This is complex, so we'll do basic fixing
    $urlPattern = '(?<![(\[<])https?://[^\s<>\)]+(?![>\)]\])'
    $urls = [regex]::Matches($content, $urlPattern)
    foreach ($match in $urls) {
        $url = $match.Value
        # Skip if URL is already in a link or code block
        $index = $match.Index
        $before = $content.Substring([Math]::Max(0, $index - 10), [Math]::Min(10, $index))
        if ($before -notmatch '[\[\(`]$') {
            $content = $content.Replace($url, "<$url>")
            $fixed++
        }
    }
    if ($urls.Count -gt 0) {
        Write-Host "  Fixed: Wrapped $($urls.Count) bare URLs" -ForegroundColor Green
    }
    
    # Fix MD012: Remove multiple consecutive blank lines
    $beforeBlank = $content
    $content = $content -replace '(?m)(\r?\n){3,}', "`n`n"
    
    # Fix MD047: Ensure file ends with newline
    if (-not $content.EndsWith("`n")) {
        $content += "`n"
        $fixed++
        Write-Host "  Fixed: Added newline at end of file" -ForegroundColor Green
    }
    
    # Write back if changed
    if ($content -ne $original) {
        if (-not $DryRun) {
            Set-Content -Path $FilePath -Value $content -Encoding UTF8 -NoNewline
        }
        return $fixed
    }
    
    return 0
}

foreach ($file in $mdFiles) {
    Write-Host "`nProcessing: $($file.Name)" -ForegroundColor Yellow
    $fixed = Fix-MarkdownFile -FilePath $file.FullName
    
    if ($fixed -gt 0) {
        $filesFixed++
        $errorsFixed += $fixed
        Write-Host "  Total fixes in file: $fixed" -ForegroundColor Cyan
    }
}

Write-Host "`n========================================" -ForegroundColor Cyan
Write-Host "SUMMARY" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Files processed: $($mdFiles.Count)" -ForegroundColor White
Write-Host "Files fixed: $filesFixed" -ForegroundColor Green
Write-Host "Total errors fixed: $errorsFixed" -ForegroundColor Green

if ($DryRun) {
    Write-Host "`nDRY RUN - No files were modified" -ForegroundColor Yellow
    Write-Host "Run without -DryRun to apply changes" -ForegroundColor Yellow
}
