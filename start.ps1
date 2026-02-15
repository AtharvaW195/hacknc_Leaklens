# Single-command startup script for Pasteguard (PowerShell)
# Starts Python screen guard API server, then Go server

$ErrorActionPreference = "Stop"

# Colors for output (PowerShell 5.1+)
function Write-ColorOutput($ForegroundColor) {
    $fc = $host.UI.RawUI.ForegroundColor
    $host.UI.RawUI.ForegroundColor = $ForegroundColor
    if ($args) {
        Write-Output $args
    }
    $host.UI.RawUI.ForegroundColor = $fc
}

# Configuration (can be overridden by env vars)
$PY_HOST = if ($env:SCREEN_GUARD_API_HOST) { $env:SCREEN_GUARD_API_HOST } else { "127.0.0.1" }
$PY_PORT = if ($env:SCREEN_GUARD_API_PORT) { $env:SCREEN_GUARD_API_PORT } else { "8081" }
$GO_HOST = if ($env:GO_HOST) { $env:GO_HOST } else { "" }
$GO_PORT = if ($env:GO_PORT) { $env:GO_PORT } else { "8080" }
$PYTHON_CMD = if ($env:PYTHON) { $env:PYTHON } else { "python" }
$GO_CMD = if ($env:GO_CMD) { $env:GO_CMD } else { "go" }

# Calculate full addresses
$PY_URL = "http://${PY_HOST}:${PY_PORT}"
$SCREEN_GUARD_BASE_URL = if ($env:SCREEN_GUARD_BASE_URL) { $env:SCREEN_GUARD_BASE_URL } else { $PY_URL }

# Create logs directory
New-Item -ItemType Directory -Force -Path "logs" | Out-Null

Write-ColorOutput Green "Starting Pasteguard services..."

# Find Python executable
try {
    $null = Get-Command $PYTHON_CMD -ErrorAction Stop
} catch {
    Write-ColorOutput Red "ERROR: $PYTHON_CMD not found. Please install Python 3 or set PYTHON env var."
    exit 1
}

# Check for virtual environment
if (Test-Path ".venv") {
    Write-ColorOutput Yellow "Found .venv, activating..."
    & ".venv\Scripts\Activate.ps1"
    $PYTHON_CMD = "python"
} elseif (Test-Path "venv") {
    Write-ColorOutput Yellow "Found venv, activating..."
    & "venv\Scripts\Activate.ps1"
    $PYTHON_CMD = "python"
}

# Check if port is already in use and kill existing process
Write-ColorOutput Yellow "Checking if port ${PY_PORT} is already in use..."
$killedProcesses = @()
try {
    $existingProcess = Get-NetTCPConnection -LocalPort $PY_PORT -ErrorAction SilentlyContinue | Select-Object -ExpandProperty OwningProcess -Unique
    if ($existingProcess) {
        Write-ColorOutput Yellow "Port ${PY_PORT} is in use by process(es): $($existingProcess -join ', ')"
        Write-ColorOutput Yellow "Stopping existing process(es)..."
        foreach ($procId in $existingProcess) {
            try {
                Stop-Process -Id $procId -Force -ErrorAction SilentlyContinue
                $killedProcesses += $procId
                Start-Sleep -Milliseconds 500
            } catch {
                Write-ColorOutput Yellow "Could not stop process $procId, continuing..."
            }
        }
    }
} catch {
    # Get-NetTCPConnection might not be available, try alternative method
    try {
        $netstat = netstat -ano | Select-String ":$PY_PORT.*LISTENING"
        if ($netstat) {
            $pids = $netstat | ForEach-Object { ($_ -split '\s+')[-1] } | Select-Object -Unique
            if ($pids) {
                Write-ColorOutput Yellow "Port ${PY_PORT} is in use. Stopping process(es): $($pids -join ', ')"
                foreach ($processId in $pids) {
                    try {
                        Stop-Process -Id $processId -Force -ErrorAction SilentlyContinue
                        $killedProcesses += $processId
                    } catch {}
                }
            }
        }
    } catch {
        Write-ColorOutput Yellow "Could not check for existing processes on port ${PY_PORT}"
    }
}

# Wait for processes to fully release resources (especially file handles and ports)
if ($killedProcesses.Count -gt 0) {
    Write-ColorOutput Yellow "Waiting for processes to release resources (ports, file handles)..."
    Start-Sleep -Seconds 3
    
    # Verify port is actually free
    $portStillInUse = $true
    $maxPortCheckAttempts = 10
    $portCheckAttempt = 0
    
    while ($portStillInUse -and $portCheckAttempt -lt $maxPortCheckAttempts) {
        Start-Sleep -Seconds 1
        $portCheckAttempt++
        
        try {
            $portCheck = Get-NetTCPConnection -LocalPort $PY_PORT -ErrorAction SilentlyContinue
            if (-not $portCheck) {
                $portStillInUse = $false
                Write-ColorOutput Green "Port ${PY_PORT} is now free!"
                break
            } else {
                $remainingPids = $portCheck | Select-Object -ExpandProperty OwningProcess -Unique
                Write-ColorOutput Yellow "Port ${PY_PORT} still in use by: $($remainingPids -join ', ') (attempt $portCheckAttempt/$maxPortCheckAttempts)"
                # Try killing again
                foreach ($procId in $remainingPids) {
                    try {
                        Stop-Process -Id $procId -Force -ErrorAction SilentlyContinue
                    } catch {}
                }
            }
        } catch {
            # Port might be free if we can't check it
            $portStillInUse = $false
            Write-ColorOutput Yellow "Could not verify port status, assuming it's free"
        }
    }
    
    if ($portStillInUse) {
        Write-ColorOutput Red "WARNING: Port ${PY_PORT} may still be in use. Starting anyway..."
    }
}

# Start Python API server in background
Write-ColorOutput Green "Starting Python Screen Guard API server on ${PY_HOST}:${PY_PORT}..."
# Use separate files for stdout and stderr (PowerShell limitation)
# Run api_server.py directly from screen_guard_service directory
$repoRoot = (Get-Location).Path
$stdoutFile = "$repoRoot\logs\backend.log"
$stderrFile = "$repoRoot\logs\backend_stderr.log"
$apiServerPath = "$repoRoot\screen_guard_service\api_server.py"

# Clear old log files (with retry if locked)
function Clear-LogFile {
    param($FilePath)
    if (Test-Path $FilePath) {
        $maxRetries = 5
        $retryCount = 0
        while ($retryCount -lt $maxRetries) {
            try {
                Clear-Content $FilePath -ErrorAction Stop
                break
            } catch {
                $retryCount++
                if ($retryCount -lt $maxRetries) {
                    Start-Sleep -Milliseconds 500
                } else {
                    # If we can't clear it, try to delete and recreate
                    try {
                        Remove-Item $FilePath -Force -ErrorAction Stop
                    } catch {
                        Write-ColorOutput Yellow "Warning: Could not clear log file $FilePath, will append to it"
                    }
                }
            }
        }
    }
}

Clear-LogFile -FilePath $stdoutFile
Clear-LogFile -FilePath $stderrFile

# Final port check right before starting
Write-ColorOutput Yellow "Final check: Verifying port ${PY_PORT} is free..."
$finalPortCheck = Get-NetTCPConnection -LocalPort $PY_PORT -ErrorAction SilentlyContinue
if ($finalPortCheck) {
    $finalPids = $finalPortCheck | Select-Object -ExpandProperty OwningProcess -Unique | Where-Object { $_ -gt 0 }
    if ($finalPids.Count -eq 0) {
        Write-ColorOutput Green "Port ${PY_PORT} appears free (no valid processes found), proceeding..."
    } else {
        Write-ColorOutput Red "WARNING: Port ${PY_PORT} is still in use by: $($finalPids -join ', ')"
        Write-ColorOutput Yellow "Attempting to kill these processes again..."
        foreach ($processId in $finalPids) {
            try {
                Stop-Process -Id $processId -Force -ErrorAction SilentlyContinue
                Write-ColorOutput Yellow "  Killed process $processId"
            } catch {
                Write-ColorOutput Red "  Could not kill process $processId"
            }
        }
        Write-ColorOutput Yellow "Waiting 2 more seconds for port to be released..."
        Start-Sleep -Seconds 2
        
        # Check one more time
        $finalPortCheck2 = Get-NetTCPConnection -LocalPort $PY_PORT -ErrorAction SilentlyContinue
        if ($finalPortCheck2) {
            $finalPids2 = $finalPortCheck2 | Select-Object -ExpandProperty OwningProcess -Unique | Where-Object { $_ -gt 0 }
            if ($finalPids2.Count -gt 0) {
                Write-ColorOutput Red "ERROR: Port ${PY_PORT} is still in use after cleanup attempts!"
                Write-ColorOutput Yellow "You may need to manually kill the process or wait a few seconds."
                Write-ColorOutput Yellow "To find and kill the process manually:"
                Write-ColorOutput Yellow "  Get-NetTCPConnection -LocalPort ${PY_PORT} | Select-Object OwningProcess"
                Write-ColorOutput Yellow "  Stop-Process -Id <PID> -Force"
                exit 1
            } else {
                Write-ColorOutput Green "Port ${PY_PORT} is now free!"
            }
        } else {
            Write-ColorOutput Green "Port ${PY_PORT} is now free!"
        }
    }
} else {
    Write-ColorOutput Green "Port ${PY_PORT} is free, proceeding..."
}

Write-ColorOutput Yellow "Starting Python process..."
Write-ColorOutput Yellow "  Command: $PYTHON_CMD $apiServerPath"
Write-ColorOutput Yellow "  Working Directory: $repoRoot\screen_guard_service"
Write-ColorOutput Yellow "  Logs: $stdoutFile"
Write-ColorOutput Yellow "  Error Logs: $stderrFile"

# Start Python process with real-time output capture
$PY_PROCESS = Start-Process -FilePath $PYTHON_CMD -ArgumentList $apiServerPath -PassThru -NoNewWindow -WorkingDirectory "$repoRoot\screen_guard_service" -RedirectStandardOutput $stdoutFile -RedirectStandardError $stderrFile

$startupId = "ps1-$(Get-Date -Format 'yyyyMMddHHmmss')-$($PY_PROCESS.Id)"
$timestamp = Get-Date -Format "yyyy-MM-ddTHH:mm:ss.fffZ"
Write-Output "$timestamp | INFO | START_PS1 | PROCESS_START | $startupId | INIT | Python process started: PID=$($PY_PROCESS.Id), Command=$PYTHON_CMD $apiServerPath"
Write-ColorOutput Green "Python process started with PID: $($PY_PROCESS.Id)"
Write-ColorOutput Yellow "Waiting 3 seconds for process to initialize and bind to port..."
Start-Sleep -Seconds 3

# Check if process is still running
if ($PY_PROCESS.HasExited) {
    $timestamp = Get-Date -Format "yyyy-MM-ddTHH:mm:ss.fffZ"
    Write-Output "$timestamp | ERROR | START_PS1 | PROCESS_START | $startupId | FAILED | Python process exited immediately! Exit code: $($PY_PROCESS.ExitCode)"
    Write-ColorOutput Red "ERROR: Python process exited immediately!"
    Write-ColorOutput Red "Exit code: $($PY_PROCESS.ExitCode)"
}

# Show initial log output
if (Test-Path $stdoutFile) {
    $initialLog = Get-Content $stdoutFile -Tail 5 -ErrorAction SilentlyContinue
    if ($initialLog) {
        Write-ColorOutput Yellow "Initial output:"
        $initialLog | ForEach-Object { Write-ColorOutput Yellow "  $_" }
    }
}
if (Test-Path $stderrFile) {
    $initialErr = Get-Content $stderrFile -Tail 5 -ErrorAction SilentlyContinue
    if ($initialErr) {
        Write-ColorOutput Red "Initial errors:"
        $initialErr | ForEach-Object { Write-ColorOutput Red "  $_" }
    }
}

# Function to cleanup on exit
function Cleanup {
    Write-ColorOutput Yellow "`nShutting down..."
    if ($PY_PROCESS -and !$PY_PROCESS.HasExited) {
        Write-ColorOutput Yellow "Stopping Python service (PID: $($PY_PROCESS.Id))..."
        Stop-Process -Id $PY_PROCESS.Id -Force -ErrorAction SilentlyContinue
    }
    Write-ColorOutput Green "Shutdown complete."
    exit 0
}

# Register cleanup on Ctrl+C
[Console]::TreatControlCAsInput = $false
$null = Register-EngineEvent PowerShell.Exiting -Action { Cleanup }

# Wait for Python API to be ready
Write-ColorOutput Yellow "Waiting for Python API to be ready..."
$MAX_WAIT = 30
$WAIT_COUNT = 0
$READY = $false

while ($WAIT_COUNT -lt $MAX_WAIT) {
    $healthCheckId = "ps1-health-$WAIT_COUNT"
    $timestamp = Get-Date -Format "yyyy-MM-ddTHH:mm:ss.fffZ"
    Write-Output "$timestamp | INFO | START_PS1 | HEALTH_CHECK | $healthCheckId | POLLING | Checking Python API health (attempt $($WAIT_COUNT + 1)/$MAX_WAIT)"
    
    try {
        $response = Invoke-WebRequest -Uri "${PY_URL}/health" -TimeoutSec 2 -ErrorAction Stop
        if ($response.StatusCode -eq 200) {
            $timestamp = Get-Date -Format "yyyy-MM-ddTHH:mm:ss.fffZ"
            Write-Output "$timestamp | INFO | START_PS1 | HEALTH_CHECK | $healthCheckId | SUCCESS | Python API is ready! Status: $($response.StatusCode)"
            Write-ColorOutput Green "Python API is ready!"
            $READY = $true
            break
        }
    } catch {
        $timestamp = Get-Date -Format "yyyy-MM-ddTHH:mm:ss.fffZ"
        Write-Output "$timestamp | DEBUG | START_PS1 | HEALTH_CHECK | $healthCheckId | WAITING | Not ready yet: $($_.Exception.Message)"
        # Not ready yet, continue waiting
    }
    
    if ($PY_PROCESS.HasExited) {
        Write-ColorOutput Red "ERROR: Python API server failed to start (process exited)."
        Write-ColorOutput Red "Exit code: $($PY_PROCESS.ExitCode)"
        Write-ColorOutput Yellow "Checking error logs..."
        
        # Show all available logs
        if (Test-Path $stderrFile) {
            $errorContent = Get-Content $stderrFile -ErrorAction SilentlyContinue
            if ($errorContent) {
                Write-ColorOutput Red "=== STDERR LOG (Last 20 lines) ==="
                $errorContent | Select-Object -Last 20 | ForEach-Object { Write-ColorOutput Red "  $_" }
            } else {
                Write-ColorOutput Yellow "  (stderr log is empty)"
            }
        } else {
            Write-ColorOutput Yellow "  (stderr log file not found)"
        }
        
        if (Test-Path $stdoutFile) {
            $stdoutContent = Get-Content $stdoutFile -ErrorAction SilentlyContinue
            if ($stdoutContent) {
                Write-ColorOutput Yellow "=== STDOUT LOG (Last 20 lines) ==="
                $stdoutContent | Select-Object -Last 20 | ForEach-Object { Write-ColorOutput Yellow "  $_" }
            } else {
                Write-ColorOutput Yellow "  (stdout log is empty)"
            }
        } else {
            Write-ColorOutput Yellow "  (stdout log file not found)"
        }
        
        Write-ColorOutput Red "`nFull logs available at:"
        Write-ColorOutput Red "  - $stdoutFile"
        Write-ColorOutput Red "  - $stderrFile"
        Write-ColorOutput Yellow "`nTo view logs in real-time, run:"
        Write-ColorOutput Yellow "  Get-Content $stdoutFile -Wait -Tail 50"
        exit 1
    }
    
    Start-Sleep -Seconds 1
    $WAIT_COUNT++
}

if (-not $READY) {
    Write-ColorOutput Red "ERROR: Python API server did not become ready within ${MAX_WAIT} seconds."
    Write-ColorOutput Yellow "Process status:"
    if ($PY_PROCESS) {
        Write-ColorOutput Yellow "  PID: $($PY_PROCESS.Id)"
        Write-ColorOutput Yellow "  HasExited: $($PY_PROCESS.HasExited)"
        if ($PY_PROCESS.HasExited) {
            Write-ColorOutput Yellow "  ExitCode: $($PY_PROCESS.ExitCode)"
        }
    }
    
    Write-ColorOutput Yellow "`nRecent logs:"
    if (Test-Path $stderrFile) {
        $recentErr = Get-Content $stderrFile -Tail 10 -ErrorAction SilentlyContinue
        if ($recentErr) {
            Write-ColorOutput Red "STDERR:"
            $recentErr | ForEach-Object { Write-ColorOutput Red "  $_" }
        }
    }
    if (Test-Path $stdoutFile) {
        $recentOut = Get-Content $stdoutFile -Tail 10 -ErrorAction SilentlyContinue
        if ($recentOut) {
            Write-ColorOutput Yellow "STDOUT:"
            $recentOut | ForEach-Object { Write-ColorOutput Yellow "  $_" }
        }
    }
    
    if ($PY_PROCESS -and !$PY_PROCESS.HasExited) {
        Stop-Process -Id $PY_PROCESS.Id -Force -ErrorAction SilentlyContinue
    }
    exit 1
}

# Check if Go server port is already in use
Write-ColorOutput Yellow "Checking if Go server port ${GO_PORT} is available..."
try {
    $goPortInUse = Get-NetTCPConnection -LocalPort $GO_PORT -ErrorAction SilentlyContinue
    if ($goPortInUse) {
        $goPids = $goPortInUse | Select-Object -ExpandProperty OwningProcess -Unique
        Write-ColorOutput Yellow "Port ${GO_PORT} is in use by process(es): $($goPids -join ', ')"
        Write-ColorOutput Yellow "Stopping existing process(es)..."
        foreach ($procId in $goPids) {
            try {
                Stop-Process -Id $procId -Force -ErrorAction SilentlyContinue
                Start-Sleep -Milliseconds 500
            } catch {
                Write-ColorOutput Yellow "Could not stop process $procId, continuing..."
            }
        }
        Start-Sleep -Seconds 1
    }
} catch {
    # Fallback to netstat if Get-NetTCPConnection not available
    try {
        $netstat = netstat -ano | Select-String ":$GO_PORT.*LISTENING"
        if ($netstat) {
            $pids = $netstat | ForEach-Object { ($_ -split '\s+')[-1] } | Select-Object -Unique
            if ($pids) {
                Write-ColorOutput Yellow "Port ${GO_PORT} is in use. Stopping process(es): $($pids -join ', ')"
                foreach ($procId in $pids) {
                    try {
                        Stop-Process -Id $procId -Force -ErrorAction SilentlyContinue
                    } catch {}
                }
                Start-Sleep -Seconds 1
            }
        }
    } catch {
        Write-ColorOutput Yellow "Could not check for existing processes on port ${GO_PORT}"
    }
}

# Export for Go server
$env:SCREEN_GUARD_BASE_URL = $SCREEN_GUARD_BASE_URL

# Start Go server in foreground
Write-ColorOutput Green "Starting Go server on :${GO_PORT}..."
Write-ColorOutput Green "Python API: ${PY_URL}"
Write-ColorOutput Green "Go Server: http://localhost:${GO_PORT}"
Write-ColorOutput Yellow "Press Ctrl+C to stop all services`n"

try {
    if ($GO_HOST) {
        & $GO_CMD run . serve --addr "${GO_HOST}:${GO_PORT}" 2>&1 | Tee-Object -FilePath "logs\go_server.log"
    } else {
        & $GO_CMD run . serve --addr ":${GO_PORT}" 2>&1 | Tee-Object -FilePath "logs\go_server.log"
    }
} catch {
    Write-ColorOutput Red "Error: $_"
} finally {
    Cleanup
}

