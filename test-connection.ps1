# test-connection.ps1
# Quick test script to verify Go and Python servers are connected

Write-Host "`n=== Testing Server Connection ===" -ForegroundColor Cyan
Write-Host ""

# Test Python API directly
Write-Host "1. Testing Python API (http://127.0.0.1:8081/health)..." -ForegroundColor Yellow
try {
    $py = Invoke-WebRequest -Uri "http://127.0.0.1:8081/health" -TimeoutSec 2 -ErrorAction Stop
    Write-Host "   ✓ Python API: OK (Status: $($py.StatusCode))" -ForegroundColor Green
    $pyContent = $py.Content | ConvertFrom-Json
    Write-Host "   Response: $($pyContent | ConvertTo-Json -Compress)" -ForegroundColor Gray
} catch {
    Write-Host "   ✗ Python API: FAILED" -ForegroundColor Red
    Write-Host "   Error: $_" -ForegroundColor Red
    Write-Host "   → Make sure Python server is running: cd screen_guard_service && python api_server.py" -ForegroundColor Yellow
}

Write-Host ""

# Test Go Server
Write-Host "2. Testing Go Server (http://localhost:8080/health)..." -ForegroundColor Yellow
try {
    $go = Invoke-WebRequest -Uri "http://localhost:8080/health" -TimeoutSec 2 -ErrorAction Stop
    Write-Host "   ✓ Go Server: OK (Status: $($go.StatusCode))" -ForegroundColor Green
} catch {
    Write-Host "   ✗ Go Server: FAILED" -ForegroundColor Red
    Write-Host "   Error: $_" -ForegroundColor Red
    Write-Host "   → Make sure Go server is running: `$env:SCREEN_GUARD_BASE_URL='http://127.0.0.1:8081'; go run . serve --addr :8080" -ForegroundColor Yellow
}

Write-Host ""

# Test Go → Python Proxy
Write-Host "3. Testing Go → Python Proxy (http://localhost:8080/api/screen-guard/health)..." -ForegroundColor Yellow
try {
    $proxy = Invoke-WebRequest -Uri "http://localhost:8080/api/screen-guard/health" -TimeoutSec 2 -ErrorAction Stop
    Write-Host "   ✓ Proxy: OK (Status: $($proxy.StatusCode))" -ForegroundColor Green
    $proxyContent = $proxy.Content | ConvertFrom-Json
    Write-Host "   Response: $($proxyContent | ConvertTo-Json -Compress)" -ForegroundColor Gray
} catch {
    Write-Host "   ✗ Proxy: FAILED" -ForegroundColor Red
    Write-Host "   Error: $_" -ForegroundColor Red
    Write-Host "   → Check that SCREEN_GUARD_BASE_URL is set correctly in Go server terminal" -ForegroundColor Yellow
}

Write-Host ""

# Test Status Endpoint
Write-Host "4. Testing Status Endpoint (http://localhost:8080/api/screen-guard/status)..." -ForegroundColor Yellow
try {
    $status = Invoke-RestMethod -Uri "http://localhost:8080/api/screen-guard/status" -TimeoutSec 2 -ErrorAction Stop
    Write-Host "   ✓ Status: OK" -ForegroundColor Green
    Write-Host "   Running: $($status.running)" -ForegroundColor Gray
    Write-Host "   PID: $($status.pid)" -ForegroundColor Gray
    if ($status.startedAt) {
        Write-Host "   Started At: $($status.startedAt)" -ForegroundColor Gray
    }
} catch {
    Write-Host "   ✗ Status: FAILED" -ForegroundColor Red
    Write-Host "   Error: $_" -ForegroundColor Red
}

Write-Host ""
Write-Host "=== Test Complete ===" -ForegroundColor Cyan
Write-Host ""

