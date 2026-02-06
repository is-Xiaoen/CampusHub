# ============================================================================
# CampusHub One-Click Deploy Script (Windows PowerShell)
# ============================================================================
#
# Usage:
#   .\deploy\scripts\deploy.ps1                    # Full deploy (build + upload + restart)
#   .\deploy\scripts\deploy.ps1 -SkipBuild         # Skip build, upload only
#   .\deploy\scripts\deploy.ps1 -Service user      # Deploy user service only
#   .\deploy\scripts\deploy.ps1 -Service activity  # Deploy activity service only
#   .\deploy\scripts\deploy.ps1 -UploadConfig      # Also upload config files
#   .\deploy\scripts\deploy.ps1 -Init              # First time: init server directories
#
# ============================================================================

param(
    [string]$Service = "all",
    [switch]$SkipBuild,
    [switch]$UploadConfig,
    [switch]$Init
)

# ==================== Configuration (modify for your environment) ====================
$ServerIP = "192.168.10.9"
$ServerUser = "root"
$ServerPath = "/opt/campushub"
# ====================================================================================

# Change to project root directory
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectRoot = Split-Path -Parent (Split-Path -Parent $ScriptDir)
Set-Location $ProjectRoot

Write-Host ""
Write-Host "========================================================" -ForegroundColor Cyan
Write-Host "  CampusHub One-Click Deploy" -ForegroundColor Cyan
Write-Host "  Server: $ServerUser@$ServerIP" -ForegroundColor Cyan
Write-Host "========================================================" -ForegroundColor Cyan
Write-Host ""

# ==================== Init Mode ====================
if ($Init) {
    Write-Host "[INIT] Creating server directories..." -ForegroundColor Yellow

    ssh "$ServerUser@$ServerIP" "mkdir -p $ServerPath/bin $ServerPath/config $ServerPath/logs $ServerPath/pids && echo 'Directories created successfully'"

    if ($LASTEXITCODE -ne 0) {
        Write-Host "[ERROR] Failed to create directories" -ForegroundColor Red
        exit 1
    }

    Write-Host ""
    Write-Host "[UPLOAD] run.sh script..." -ForegroundColor Yellow
    scp "deploy/server/run.sh" "${ServerUser}@${ServerIP}:${ServerPath}/"

    if ($LASTEXITCODE -ne 0) {
        Write-Host "[ERROR] Failed to upload run.sh" -ForegroundColor Red
        exit 1
    }

    ssh "$ServerUser@$ServerIP" "chmod +x $ServerPath/run.sh"

    Write-Host ""
    Write-Host "[UPLOAD] Config files..." -ForegroundColor Yellow
    scp deploy/docker/config/*.yaml "${ServerUser}@${ServerIP}:${ServerPath}/config/"

    if ($LASTEXITCODE -ne 0) {
        Write-Host "[ERROR] Failed to upload config files" -ForegroundColor Red
        exit 1
    }

    Write-Host ""
    Write-Host "========================================================" -ForegroundColor Green
    Write-Host "  Init Complete!" -ForegroundColor Green
    Write-Host "" -ForegroundColor Green
    Write-Host "  Next step: .\deploy\scripts\deploy.ps1" -ForegroundColor Green
    Write-Host "========================================================" -ForegroundColor Green
    exit 0
}

# ==================== Step 1: Build ====================
if (!$SkipBuild) {
    Write-Host "[STEP 1/3] Building services..." -ForegroundColor Cyan
    Write-Host ""

    & "$ScriptDir\build.ps1" -Service $Service

    if ($LASTEXITCODE -ne 0) {
        Write-Host "Build failed, aborting deploy" -ForegroundColor Red
        exit 1
    }
} else {
    Write-Host "[STEP 1/3] Skipping build (using existing binaries)" -ForegroundColor Gray
}

# ==================== Step 2: Upload ====================
Write-Host ""
Write-Host "[STEP 2/3] Uploading to server..." -ForegroundColor Cyan
Write-Host ""

# Determine which files to upload
$FilesToUpload = @()
switch ($Service) {
    "user" {
        $FilesToUpload = @("user-api", "user-rpc")
    }
    "activity" {
        $FilesToUpload = @("activity-api", "activity-rpc")
    }
    "chat" {
        $FilesToUpload = @("chat-api", "chat-rpc")
    }
    "all" {
        $FilesToUpload = @("user-api", "user-rpc", "activity-api", "activity-rpc", "chat-api", "chat-rpc")
    }
}

# Upload binary files
foreach ($File in $FilesToUpload) {
    $LocalFile = "deploy\bin\$File"
    if (Test-Path $LocalFile) {
        $FileInfo = Get-Item $LocalFile
        $Size = [math]::Round($FileInfo.Length / 1MB, 1)
        Write-Host "  [UPLOAD] $File ($Size MB)..." -ForegroundColor Yellow -NoNewline
        scp $LocalFile "${ServerUser}@${ServerIP}:${ServerPath}/bin/" 2>$null
        if ($LASTEXITCODE -eq 0) {
            Write-Host " OK" -ForegroundColor Green
        } else {
            Write-Host " FAILED" -ForegroundColor Red
        }
    } else {
        Write-Host "  [SKIP] $File (file not found)" -ForegroundColor Gray
    }
}

# Upload config files (optional)
if ($UploadConfig) {
    Write-Host ""
    Write-Host "  [UPLOAD] Config files..." -ForegroundColor Yellow
    scp deploy/docker/config/*.yaml "${ServerUser}@${ServerIP}:${ServerPath}/config/"
}

# ==================== Step 3: Restart ====================
Write-Host ""
Write-Host "[STEP 3/3] Restarting services..." -ForegroundColor Cyan
Write-Host ""

# Set permissions and restart
ssh "$ServerUser@$ServerIP" "cd $ServerPath && chmod +x bin/* && ./run.sh restart $Service"

Write-Host ""
Write-Host "========================================================" -ForegroundColor Green
Write-Host "  Deploy Complete!" -ForegroundColor Green
Write-Host "========================================================" -ForegroundColor Green
Write-Host ""
Write-Host "  Check status: ssh $ServerUser@$ServerIP '$ServerPath/run.sh status'" -ForegroundColor Cyan
Write-Host "  View logs:    ssh $ServerUser@$ServerIP '$ServerPath/run.sh logs user-api'" -ForegroundColor Cyan
Write-Host ""
