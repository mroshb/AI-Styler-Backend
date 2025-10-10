# AI Styler Windows Service Installation Script

# Check if running as administrator
if (-NOT ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")) {
    Write-Error "This script requires Administrator privileges. Please run as Administrator."
    exit 1
}

# Configuration
$ServiceName = "AI Styler Service"
$ServiceDisplayName = "AI Styler API Service"
$ServiceDescription = "AI Styler API Service for image processing and conversion"
$AppPath = "C:\inetpub\wwwroot\ai-styler"
$ExePath = "$AppPath\ai-styler.exe"
$LogPath = "$AppPath\logs"

# Create application directory
if (!(Test-Path $AppPath)) {
    New-Item -ItemType Directory -Path $AppPath -Force
    Write-Host "Created application directory: $AppPath"
}

# Create logs directory
if (!(Test-Path $LogPath)) {
    New-Item -ItemType Directory -Path $LogPath -Force
    Write-Host "Created logs directory: $LogPath"
}

# Create uploads directory
$UploadsPath = "$AppPath\uploads"
if (!(Test-Path $UploadsPath)) {
    New-Item -ItemType Directory -Path $UploadsPath -Force
    Write-Host "Created uploads directory: $UploadsPath"
}

# Stop and remove existing service if it exists
$existingService = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
if ($existingService) {
    Write-Host "Stopping existing service..."
    Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue
    Write-Host "Removing existing service..."
    sc.exe delete $ServiceName
    Start-Sleep -Seconds 2
}

# Create Windows Service
Write-Host "Creating Windows Service..."
$serviceArgs = @(
    "create"
    $ServiceName
    "binPath= `"$ExePath`""
    "DisplayName= `"$ServiceDisplayName`""
    "Description= `"$ServiceDescription`""
    "start= auto"
    "obj= LocalSystem"
)

& sc.exe $serviceArgs

if ($LASTEXITCODE -ne 0) {
    Write-Error "Failed to create Windows Service. Exit code: $LASTEXITCODE"
    exit 1
}

# Configure service recovery options
Write-Host "Configuring service recovery options..."
& sc.exe failure $ServiceName reset= 86400 actions= restart/5000/restart/5000/restart/5000

# Set service to start automatically
Write-Host "Setting service to start automatically..."
Set-Service -Name $ServiceName -StartupType Automatic

# Grant necessary permissions
Write-Host "Setting permissions..."
icacls $AppPath /grant "IIS_IUSRS:(OI)(CI)F" /T
icacls $LogPath /grant "IIS_IUSRS:(OI)(CI)F" /T
icacls $UploadsPath /grant "IIS_IUSRS:(OI)(CI)F" /T

# Create environment file
$envFile = "$AppPath\.env"
$envContent = @"
# AI Styler Production Environment Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=styler_user
DB_PASSWORD=styler_password
DB_NAME=styler
DB_SSLMODE=disable

REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

JWT_SECRET=your-production-secret-key-change-this
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=720h

SMS_PROVIDER=mock
SMS_API_KEY=
SMS_TEMPLATE_ID=100000

BCRYPT_COST=12
ARGON2_MEMORY=65536
ARGON2_ITERATIONS=3
ARGON2_PARALLELISM=2
ARGON2_SALT_LENGTH=16
ARGON2_KEY_LENGTH=32

RATE_LIMIT_OTP_PER_PHONE=3
RATE_LIMIT_OTP_PER_IP=100
RATE_LIMIT_LOGIN_PER_PHONE=5
RATE_LIMIT_LOGIN_PER_IP=10
RATE_LIMIT_WINDOW=1h

UPLOAD_MAX_SIZE=50MB
STORAGE_PATH=./uploads
SIGNED_URL_TTL=1h

TELEGRAM_BOT_TOKEN=
TELEGRAM_CHAT_ID=
LOG_LEVEL=info
SENTRY_DSN=
ENVIRONMENT=production
VERSION=1.0.0
HEALTH_ENABLED=true
GIN_MODE=release
HTTP_ADDR=:8080
"@

Set-Content -Path $envFile -Value $envContent -Encoding UTF8
Write-Host "Created environment file: $envFile"

# Start the service
Write-Host "Starting service..."
Start-Service -Name $ServiceName

# Wait for service to start
Start-Sleep -Seconds 5

# Check service status
$service = Get-Service -Name $ServiceName
if ($service.Status -eq "Running") {
    Write-Host "Service started successfully!"
    Write-Host "Service Name: $ServiceName"
    Write-Host "Status: $($service.Status)"
    Write-Host "Start Type: $($service.StartType)"
} else {
    Write-Warning "Service may not have started properly. Status: $($service.Status)"
}

# Test health endpoint
Write-Host "Testing health endpoint..."
try {
    $response = Invoke-WebRequest -Uri "http://localhost:8080/health/" -TimeoutSec 10
    if ($response.StatusCode -eq 200) {
        Write-Host "Health check passed! Service is responding correctly."
    } else {
        Write-Warning "Health check returned status code: $($response.StatusCode)"
    }
} catch {
    Write-Warning "Health check failed: $($_.Exception.Message)"
    Write-Host "Please check the service logs for more information."
}

Write-Host ""
Write-Host "Installation completed!"
Write-Host "Service Name: $ServiceName"
Write-Host "Application Path: $AppPath"
Write-Host "Logs Path: $LogPath"
Write-Host ""
Write-Host "Useful commands:"
Write-Host "  Start Service:   Start-Service -Name `"$ServiceName`""
Write-Host "  Stop Service:    Stop-Service -Name `"$ServiceName`""
Write-Host "  Restart Service: Restart-Service -Name `"$ServiceName`""
Write-Host "  Service Status:  Get-Service -Name `"$ServiceName`""
Write-Host "  View Logs:       Get-EventLog -LogName Application -Source `"$ServiceName`" -Newest 50"
