# Generate Secure Credentials for JAX Trading Assistant
# This script generates cryptographically secure random passwords and creates/updates .env file

function Generate-SecurePassword {
    param (
        [int]$Length = 32
    )
    
    # Use cryptographic RNG for secure password generation
    $bytes = New-Object byte[] $Length
    $rng = [System.Security.Cryptography.RandomNumberGenerator]::Create()
    $rng.GetBytes($bytes)
    
    # Convert to base64 and clean up for use in environment variables
    $password = [Convert]::ToBase64String($bytes)
    # Remove characters that might cause issues in environment variables
    $password = $password -replace '[/+=]', ''
    
    return $password.Substring(0, [Math]::Min($Length, $password.Length))
}

# Script configuration
$EnvFile = Join-Path $PSScriptRoot "..\..env"
$EnvExampleFile = Join-Path $PSScriptRoot "..\.env.example"

Write-Host "================================================" -ForegroundColor Cyan
Write-Host "  JAX Trading Assistant - Credential Generator" -ForegroundColor Cyan
Write-Host "================================================" -ForegroundColor Cyan
Write-Host ""

# Check if .env already exists
if (Test-Path $EnvFile) {
    Write-Host "WARNING: .env file already exists!" -ForegroundColor Yellow
    $response = Read-Host "Do you want to regenerate credentials? This will overwrite existing values. (yes/no)"
    
    if ($response -ne "yes") {
        Write-Host "Operation cancelled. Existing .env file preserved." -ForegroundColor Green
        exit 0
    }
}

# Generate secure credentials
Write-Host "Generating cryptographically secure credentials..." -ForegroundColor Green

$PostgresPassword = Generate-SecurePassword -Length 32
$RedisPassword = Generate-SecurePassword -Length 32
$JwtSecret = Generate-SecurePassword -Length 64

# Create .env file content
$EnvContent = @"
# JAX Trading Assistant - Environment Configuration
# Generated: $(Get-Date -Format "yyyy-MM-dd HH:mm:ss")
# SECURITY WARNING: Keep this file secure and never commit to version control!

# Database Configuration
POSTGRES_USER=jax
POSTGRES_PASSWORD=$PostgresPassword
POSTGRES_DB=jax
DATABASE_URL=postgresql://jax:$PostgresPassword@postgres:5432/jax
KNOWLEDGE_DATABASE_URL=postgresql://jax:$PostgresPassword@postgres:5432/jax_knowledge

# Redis Configuration (optional - for future use)
REDIS_PASSWORD=$RedisPassword
REDIS_URL=redis://:$RedisPassword@redis:6379/0

# JWT Authentication
JWT_SECRET=$JwtSecret
JWT_EXPIRY=24h
JWT_REFRESH_EXPIRY=168h

# CORS Configuration
# Comma-separated list of allowed origins
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173,http://127.0.0.1:3000,http://127.0.0.1:5173
CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE,OPTIONS
CORS_ALLOWED_HEADERS=Content-Type,Authorization
CORS_ALLOW_CREDENTIALS=true

# Rate Limiting Configuration
RATE_LIMIT_REQUESTS_PER_MINUTE=100
RATE_LIMIT_REQUESTS_PER_HOUR=1000
RATE_LIMIT_ENABLED=true

# API Configuration
API_PORT=8081
API_HOST=0.0.0.0

# Default Admin Credentials (change after first login!)
ADMIN_USERNAME=admin
ADMIN_PASSWORD_HASH=

# Application Environment
ENVIRONMENT=development
LOG_LEVEL=info
"@

# Write .env file
try {
    $EnvContent | Out-File -FilePath $EnvFile -Encoding UTF8 -Force
    Write-Host ""
    Write-Host "✓ .env file created successfully!" -ForegroundColor Green
    Write-Host "  Location: $EnvFile" -ForegroundColor Gray
}
catch {
    Write-Host "ERROR: Failed to create .env file: $_" -ForegroundColor Red
    exit 1
}

# Display generated credentials summary
Write-Host ""
Write-Host "================================================" -ForegroundColor Cyan
Write-Host "  Generated Credentials Summary" -ForegroundColor Cyan
Write-Host "================================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "PostgreSQL:" -ForegroundColor Yellow
Write-Host "  Username: jax" -ForegroundColor Gray
Write-Host "  Password: $($PostgresPassword.Substring(0, 8))..." -ForegroundColor Gray
Write-Host ""
Write-Host "JWT Secret:" -ForegroundColor Yellow
Write-Host "  Secret: $($JwtSecret.Substring(0, 16))..." -ForegroundColor Gray
Write-Host ""
Write-Host "Redis Password:" -ForegroundColor Yellow
Write-Host "  Password: $($RedisPassword.Substring(0, 8))..." -ForegroundColor Gray
Write-Host ""

# Display security instructions
Write-Host "================================================" -ForegroundColor Cyan
Write-Host "  Security Instructions" -ForegroundColor Cyan
Write-Host "================================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "1. IMPORTANT: Add .env to your .gitignore file" -ForegroundColor Yellow
Write-Host "2. Store credentials in a secure password manager" -ForegroundColor Yellow
Write-Host "3. Never commit .env to version control" -ForegroundColor Yellow
Write-Host "4. Set appropriate file permissions:" -ForegroundColor Yellow
Write-Host "   - Windows: Right-click .env > Properties > Security" -ForegroundColor Gray
Write-Host "   - Ensure only your user account has access" -ForegroundColor Gray
Write-Host ""
Write-Host "5. For production deployment:" -ForegroundColor Yellow
Write-Host "   - Use environment-specific credential management" -ForegroundColor Gray
Write-Host "   - Consider using Azure Key Vault or similar" -ForegroundColor Gray
Write-Host "   - Rotate credentials regularly" -ForegroundColor Gray
Write-Host ""

# Check .gitignore
$GitignoreFile = Join-Path $PSScriptRoot "..\.gitignore"
if (Test-Path $GitignoreFile) {
    $gitignoreContent = Get-Content $GitignoreFile -Raw
    if ($gitignoreContent -notmatch '\.env$') {
        Write-Host "WARNING: .env not found in .gitignore!" -ForegroundColor Red
        $addToGitignore = Read-Host "Add .env to .gitignore now? (yes/no)"
        
        if ($addToGitignore -eq "yes") {
            Add-Content -Path $GitignoreFile -Value "`n# Environment variables`n.env"
            Write-Host "✓ Added .env to .gitignore" -ForegroundColor Green
        }
    } else {
        Write-Host "✓ .env is already in .gitignore" -ForegroundColor Green
    }
}

Write-Host ""
Write-Host "================================================" -ForegroundColor Cyan
Write-Host "  Next Steps" -ForegroundColor Cyan
Write-Host "================================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "1. Review the generated .env file" -ForegroundColor White
Write-Host "2. Update CORS_ALLOWED_ORIGINS if needed" -ForegroundColor White
Write-Host "3. Run: docker-compose down -v (to remove old data)" -ForegroundColor White
Write-Host "4. Run: docker-compose up -d (to start with new credentials)" -ForegroundColor White
Write-Host ""
Write-Host "Credentials generated successfully! 🔐" -ForegroundColor Green
Write-Host ""
