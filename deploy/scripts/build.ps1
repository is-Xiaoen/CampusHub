# ============================================================================
# CampusHub Build Script (Windows PowerShell)
# ============================================================================
#
# Usage:
#   .\deploy\scripts\build.ps1           # Build all services
#   .\deploy\scripts\build.ps1 user      # Build user service only
#   .\deploy\scripts\build.ps1 activity  # Build activity service only
#   .\deploy\scripts\build.ps1 chat      # Build chat service only
#
# ============================================================================

param(
    [string]$Service = "all"
)

# Change to project root directory
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectRoot = Split-Path -Parent (Split-Path -Parent $ScriptDir)
Set-Location $ProjectRoot

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  CampusHub Cross-Compile (Linux/amd64)" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Set cross-compile environment variables (temporary, does not affect system)
$env:GOOS = "linux"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"

# Output directory
$OutputDir = "deploy\bin"

# Create output directory if not exists
if (!(Test-Path $OutputDir)) {
    New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null
    Write-Host "[CREATE] $OutputDir directory" -ForegroundColor Gray
}

# Build function
function Build-Service {
    param(
        [string]$Name,
        [string]$Type,
        [string]$Path
    )

    $Output = Join-Path $OutputDir "$Name-$Type"
    Write-Host "[BUILD] $Name-$Type ... " -ForegroundColor Yellow -NoNewline

    $StartTime = Get-Date

    # Execute build
    $ErrorOutput = $null
    go build -ldflags="-s -w" -o $Output $Path 2>&1 | ForEach-Object { $ErrorOutput += $_ }

    $EndTime = Get-Date
    $Duration = [math]::Round(($EndTime - $StartTime).TotalSeconds, 1)

    if ($LASTEXITCODE -eq 0 -and (Test-Path $Output)) {
        $FileInfo = Get-Item $Output
        $Size = [math]::Round($FileInfo.Length / 1MB, 1)
        Write-Host "OK ($Size MB, ${Duration}s)" -ForegroundColor Green
        return $true
    } else {
        Write-Host "FAILED" -ForegroundColor Red
        if ($ErrorOutput) {
            Write-Host $ErrorOutput -ForegroundColor Red
        }
        return $false
    }
}

# Track success
$AllSuccess = $true

# Build services based on parameter
switch ($Service) {
    "user" {
        if (!(Build-Service -Name "user" -Type "rpc" -Path "./app/user/rpc/user.go")) { $AllSuccess = $false }
        if (!(Build-Service -Name "user" -Type "api" -Path "./app/user/api/user.go")) { $AllSuccess = $false }
    }
    "activity" {
        if (!(Build-Service -Name "activity" -Type "rpc" -Path "./app/activity/rpc/activity.go")) { $AllSuccess = $false }
        if (!(Build-Service -Name "activity" -Type "api" -Path "./app/activity/api/activity.go")) { $AllSuccess = $false }
    }
    "chat" {
        if (!(Build-Service -Name "chat" -Type "rpc" -Path "./app/chat/rpc/chat.go")) { $AllSuccess = $false }
        if (!(Build-Service -Name "chat" -Type "api" -Path "./app/chat/api/chat.go")) { $AllSuccess = $false }
    }
    "all" {
        if (!(Build-Service -Name "user" -Type "rpc" -Path "./app/user/rpc/user.go")) { $AllSuccess = $false }
        if (!(Build-Service -Name "user" -Type "api" -Path "./app/user/api/user.go")) { $AllSuccess = $false }
        if (!(Build-Service -Name "activity" -Type "rpc" -Path "./app/activity/rpc/activity.go")) { $AllSuccess = $false }
        if (!(Build-Service -Name "activity" -Type "api" -Path "./app/activity/api/activity.go")) { $AllSuccess = $false }
        if (!(Build-Service -Name "chat" -Type "rpc" -Path "./app/chat/rpc/chat.go")) { $AllSuccess = $false }
        if (!(Build-Service -Name "chat" -Type "api" -Path "./app/chat/api/chat.go")) { $AllSuccess = $false }
    }
    default {
        Write-Host "Unknown service: $Service" -ForegroundColor Red
        Write-Host "Available: user, activity, chat, all" -ForegroundColor Yellow
        exit 1
    }
}

# Show results
Write-Host ""
if ($AllSuccess) {
    Write-Host "========================================" -ForegroundColor Green
    Write-Host "  Build Successful!" -ForegroundColor Green
    Write-Host "========================================" -ForegroundColor Green
    Write-Host ""
    Write-Host "Generated files:" -ForegroundColor Cyan

    $TotalSize = 0
    $Files = Get-ChildItem -Path $OutputDir -File -ErrorAction SilentlyContinue

    if ($Files) {
        foreach ($File in $Files) {
            $Size = [math]::Round($File.Length / 1MB, 1)
            $TotalSize += $File.Length
            $FileName = $File.Name.PadRight(20)
            Write-Host "  $FileName $Size MB" -ForegroundColor White
        }
        Write-Host "  ----------------------------------------"
        $TotalMB = [math]::Round($TotalSize / 1MB, 1)
        Write-Host "  Total:               $TotalMB MB" -ForegroundColor Yellow
    }
    Write-Host ""
} else {
    Write-Host "========================================" -ForegroundColor Red
    Write-Host "  Build Failed!" -ForegroundColor Red
    Write-Host "========================================" -ForegroundColor Red
    exit 1
}
