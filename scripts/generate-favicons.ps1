#!/usr/bin/env pwsh
# Generate Favicon Files from SVG Logo
# 
# This script converts the JAX AI Trader logo (SVG) to various favicon formats
# required for modern web applications and mobile devices.
#
# Requirements:
#   - Node.js and npm installed
#   - sharp package (will be installed if missing)
#
# Usage: .\scripts\generate-favicons.ps1

param(
    [string]$SourceSvg = "frontend/src/images/jax_ai_trader.svg",
    [string]$OutputDir = "frontend/public"
)

$ErrorActionPreference = "Stop"

Write-Host "JAX Trading Assistant - Favicon Generator" -ForegroundColor Cyan
Write-Host "=============================================" -ForegroundColor Cyan
Write-Host ""

# Check if source SVG exists
$projectRoot = Split-Path -Parent $PSScriptRoot
$sourcePath = Join-Path $projectRoot $SourceSvg

if (-not (Test-Path $sourcePath)) {
    Write-Error "Source SVG not found at: $sourcePath"
    exit 1
}

Write-Host "Source SVG found: $sourcePath" -ForegroundColor Green

# Create output directory if it doesn't exist
$outputPath = Join-Path $projectRoot $OutputDir
if (-not (Test-Path $outputPath)) {
    New-Item -ItemType Directory -Path $outputPath -Force | Out-Null
    Write-Host "Created output directory: $outputPath" -ForegroundColor Green
} else {
    Write-Host "Output directory exists: $outputPath" -ForegroundColor Green
}

# Check if Node.js is installed
try {
    $nodeVersion = node --version 2>$null
    Write-Host "Node.js installed: $nodeVersion" -ForegroundColor Green
} catch {
    Write-Error "Node.js is not installed. Please install Node.js from https://nodejs.org/"
    exit 1
}

# Create a temporary Node.js script to generate favicons
$scriptPath = Join-Path $projectRoot "temp-generate-favicons.js"

# Write the Node.js script (using here-string with single quotes to avoid interpolation)
$nodeScriptContent = @'
const sharp = require('sharp');
const fs = require('fs');
const path = require('path');

const sizes = [
    { name: 'favicon-16x16.png', size: 16 },
    { name: 'favicon-32x32.png', size: 32 },
    { name: 'favicon-192x192.png', size: 192 },
    { name: 'favicon-512x512.png', size: 512 },
    { name: 'apple-touch-icon.png', size: 180 }
];

const svgPath = process.argv[2];
const outputDir = process.argv[3];

console.log('Converting SVG to PNG favicons...\n');

async function generateFavicons() {
    const svgBuffer = fs.readFileSync(svgPath);
    
    for (const item of sizes) {
        const outputPath = path.join(outputDir, item.name);
        
        try {
            await sharp(svgBuffer)
                .resize(item.size, item.size, {
                    fit: 'contain',
                    background: { r: 0, g: 0, b: 0, alpha: 0 }
                })
                .png()
                .toFile(outputPath);
            
            console.log('Generated:', item.name, '-', item.size + 'x' + item.size);
        } catch (error) {
            console.error('Failed to generate', item.name, ':', error.message);
            process.exit(1);
        }
    }
    
    // Generate ICO file (multi-resolution)
    console.log('\nGenerating favicon.ico...');
    try {
        const icoPath = path.join(outputDir, 'favicon.ico');
        await sharp(svgBuffer)
            .resize(32, 32, {
                fit: 'contain',
                background: { r: 0, g: 0, b: 0, alpha: 0 }
            })
            .png()
            .toFile(icoPath);
        
        console.log('Generated: favicon.ico - 32x32');
        console.log('\nNote: For true ICO format with multiple resolutions, use ImageMagick');
    } catch (error) {
        console.error('Failed to generate favicon.ico:', error.message);
        process.exit(1);
    }
    
    console.log('\nAll favicons generated successfully!');
}

generateFavicons().catch(error => {
    console.error('Error:', error);
    process.exit(1);
});
'@

try {
    Write-Host ""
    Write-Host "Generating favicon files..." -ForegroundColor Yellow
    Write-Host ""
    
    # Write the Node.js script
    Set-Content -Path $scriptPath -Value $nodeScriptContent -Encoding UTF8
    
    # Install sharp temporarily if needed
    $tempDir = Join-Path $projectRoot "temp-favicon-gen"
    if (-not (Test-Path $tempDir)) {
        New-Item -ItemType Directory -Path $tempDir -Force | Out-Null
    }
    
    Push-Location $tempDir
    try {
        # Create a minimal package.json
        @{ name = "favicon-gen"; version = "1.0.0" } | ConvertTo-Json | Set-Content "package.json"
        
        Write-Host "Installing sharp package..." -ForegroundColor Yellow
        npm install --silent sharp 2>&1 | Out-Null
        
        if ($LASTEXITCODE -ne 0) {
            throw "Failed to install sharp package"
        }
        
        Write-Host "sharp package installed" -ForegroundColor Green
        Write-Host ""
        
        # Copy node_modules to project root for the script to find
        $nodeModulesSource = Join-Path $tempDir "node_modules"
        $nodeModulesDest = Join-Path $projectRoot "node_modules"
        if (-not (Test-Path $nodeModulesDest)) {
            Copy-Item $nodeModulesSource $nodeModulesDest -Recurse -Force
        } else {
            # Merge sharp into existing node_modules
            $sharpSource = Join-Path $nodeModulesSource "sharp"
            $sharpDest = Join-Path $nodeModulesDest "sharp"
            if (-not (Test-Path $sharpDest)) {
                Copy-Item $sharpSource $sharpDest -Recurse -Force
            }
        }
        
        # Run the generation script
        node $scriptPath $sourcePath $outputPath
        
        if ($LASTEXITCODE -ne 0) {
            throw "Favicon generation failed"
        }
    } finally {
        Pop-Location
    }
    
    # Clean up
    Remove-Item $scriptPath -Force -ErrorAction SilentlyContinue
    Remove-Item $tempDir -Recurse -Force -ErrorAction SilentlyContinue
    
    Write-Host ""
    Write-Host "Favicon generation complete!" -ForegroundColor Green
    Write-Host ""
    Write-Host "Generated files in ${OutputDir}:" -ForegroundColor Cyan
    Write-Host "  - favicon.ico (32x32)" -ForegroundColor White
    Write-Host "  - favicon-16x16.png" -ForegroundColor White
    Write-Host "  - favicon-32x32.png" -ForegroundColor White
    Write-Host "  - favicon-192x192.png (Android Chrome)" -ForegroundColor White
    Write-Host "  - favicon-512x512.png (Android Chrome)" -ForegroundColor White
    Write-Host "  - apple-touch-icon.png (180x180, iOS)" -ForegroundColor White
    Write-Host ""
    Write-Host "All favicon files are ready to use!" -ForegroundColor Green
    Write-Host ""
    
} catch {
    Write-Host ""
    Write-Error "Failed to generate favicons: $_"
    Write-Host ""
    Write-Host "Manual Alternative:" -ForegroundColor Yellow
    Write-Host "-------------------" -ForegroundColor Yellow
    Write-Host "You can manually convert the SVG using one of these methods:" -ForegroundColor White
    Write-Host ""
    Write-Host "1. Online Tools:" -ForegroundColor Cyan
    Write-Host "   - https://realfavicongenerator.net/" -ForegroundColor White
    Write-Host "   - https://favicon.io/" -ForegroundColor White
    Write-Host ""
    Write-Host "2. ImageMagick (if installed):" -ForegroundColor Cyan
    Write-Host "   convert -background none $SourceSvg -resize 16x16 favicon-16x16.png" -ForegroundColor White
    Write-Host "   convert -background none $SourceSvg -resize 32x32 favicon-32x32.png" -ForegroundColor White
    Write-Host ""
    exit 1
}
