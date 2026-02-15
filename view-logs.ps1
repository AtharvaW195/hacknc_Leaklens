# View logs in real-time
# Usage: .\view-logs.ps1 [python|go|all]

param(
    [string]$Service = "all"
)

$pythonLog = "logs\screen_guard_service.log"
$pythonErrLog = "logs\screen_guard_service_stderr.log"
$goLog = "logs\go_server.log"

function Show-Log {
    param($LogFile, $Name, $Color = "White")
    if (Test-Path $LogFile) {
        Write-Host "`n=== $Name ===" -ForegroundColor $Color
        Get-Content $LogFile -Wait -Tail 50
    } else {
        Write-Host "Log file not found: $LogFile" -ForegroundColor Yellow
    }
}

Write-Host "Viewing logs (Press Ctrl+C to stop)..." -ForegroundColor Green
Write-Host "Log files:" -ForegroundColor Yellow
Write-Host "  Python: $pythonLog" -ForegroundColor Yellow
Write-Host "  Python Errors: $pythonErrLog" -ForegroundColor Yellow
Write-Host "  Go: $goLog" -ForegroundColor Yellow

if ($Service -eq "python") {
    if (Test-Path $pythonErrLog) {
        Get-Content $pythonErrLog -Wait -Tail 50
    } else {
        Get-Content $pythonLog -Wait -Tail 50
    }
} elseif ($Service -eq "go") {
    Show-Log -LogFile $goLog -Name "Go Server" -Color "Cyan"
} else {
    # Show all logs
    Write-Host "`nShowing last 20 lines of each log:" -ForegroundColor Green
    
    if (Test-Path $pythonLog) {
        Write-Host "`n=== Python Server (stdout) ===" -ForegroundColor Green
        Get-Content $pythonLog -Tail 20
    }
    
    if (Test-Path $pythonErrLog) {
        Write-Host "`n=== Python Server (stderr) ===" -ForegroundColor Red
        Get-Content $pythonErrLog -Tail 20
    }
    
    if (Test-Path $goLog) {
        Write-Host "`n=== Go Server ===" -ForegroundColor Cyan
        Get-Content $goLog -Tail 20
    }
    
    Write-Host "`nTo view logs in real-time, run:" -ForegroundColor Yellow
    Write-Host "  Get-Content $pythonLog -Wait -Tail 50" -ForegroundColor White
    Write-Host "  Get-Content $pythonErrLog -Wait -Tail 50" -ForegroundColor White
    Write-Host "  Get-Content $goLog -Wait -Tail 50" -ForegroundColor White
}

