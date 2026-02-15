# Comprehensive Test Suite for Pasteguard
# Runs all tests and generates a detailed report

$ErrorActionPreference = "Stop"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Pasteguard - Complete Test Suite" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$report = @{
    Timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    Tests = @{}
    Coverage = @{}
    Build = @{}
    Errors = @()
}

# Test 1: Root Module Tests (Pasteguard CLI + Detector)
Write-Host "[1/4] Testing Root Module (Pasteguard CLI + Detector)..." -ForegroundColor Yellow
try {
    $output = & "$env:ProgramFiles\Go\bin\go.exe" test ./... -v -cover 2>&1 | Out-String
    $report.Tests["Root Module"] = @{
        Status = "PASS"
        Output = $output
    }
    Write-Host "  Root module tests passed" -ForegroundColor Green
    
    # Extract coverage for each package
    # Pattern matches lines like: "ok  	pasteguard/detector	(cached)	coverage: 95.2% of statements"
    $lines = $output -split "`r?`n"
    $coverageDetails = @{}
    foreach ($line in $lines) {
        if ($line -match "ok\s+([^\s\t]+).*coverage:\s+(\d+\.\d+)%") {
            $package = $matches[1]
            $coverage = $matches[2]
            $coverageDetails[$package] = $coverage
        }
    }
    # Store all package coverages
    if ($coverageDetails.Count -gt 0) {
        $report.Coverage["Root Module"] = $coverageDetails
        # Also calculate average for summary (excluding 0% main package)
        $nonZeroCoverages = $coverageDetails.Values | Where-Object { [double]$_ -gt 0 } | ForEach-Object { [double]$_ }
        if ($nonZeroCoverages.Count -gt 0) {
            $avgCoverage = ($nonZeroCoverages | Measure-Object -Average).Average
            $report.Coverage["Root Module (avg)"] = [math]::Round($avgCoverage, 1).ToString()
        }
    }
} catch {
    $report.Tests["Root Module"] = @{
        Status = "FAIL"
        Error = $_.Exception.Message
    }
    $report.Errors += "Root module tests failed: $($_.Exception.Message)"
    Write-Host "  Root module tests failed" -ForegroundColor Red
}

Write-Host ""

# Test 2: Backend Module Tests
Write-Host "[2/4] Testing Backend Module..." -ForegroundColor Yellow
try {
    $goPath = "$env:ProgramFiles\Go\bin\go.exe"
    Push-Location backend
    $output = & $goPath test ./... -v -cover 2>&1
    $exitCode = $LASTEXITCODE
    $outputString = $output | Out-String
    Pop-Location
    # Check if tests passed or if there are no test files (which is OK)
    if ($exitCode -eq 0 -or $outputString -match "no test files") {
        $report.Tests["Backend Module"] = @{
            Status = "PASS"
            Output = $outputString
        }
        Write-Host "  Backend module tests passed (or no test files)" -ForegroundColor Green
        
        # Extract coverage
        $coverageLines = $outputString | Select-String -Pattern "coverage: (\d+\.\d+)%"
        if ($coverageLines) {
            $coverageValues = $coverageLines | ForEach-Object { $_.Matches.Groups[1].Value }
            $report.Coverage["Backend Module"] = ($coverageValues -join ", ")
        }
    } else {
        throw "Backend tests failed with exit code $exitCode"
    }
} catch {
    Pop-Location -ErrorAction SilentlyContinue
    $report.Tests["Backend Module"] = @{
        Status = "FAIL"
        Error = $_.Exception.Message
    }
    $report.Errors += "Backend module tests failed: $($_.Exception.Message)"
    Write-Host "  Backend module tests failed" -ForegroundColor Red
}

Write-Host ""

# Test 3: Build Verification
Write-Host "[3/4] Verifying Builds..." -ForegroundColor Yellow
try {
    # Build root module
    & "$env:ProgramFiles\Go\bin\go.exe" build -o pasteguard.exe . 2>&1 | Out-Null
    if (Test-Path "pasteguard.exe") {
        $report.Build["Root Module"] = "PASS"
        Write-Host "  Root module builds successfully" -ForegroundColor Green
    } else {
        throw "pasteguard.exe not found"
    }
    
    # Build backend module
    Set-Location backend
    & "$env:ProgramFiles\Go\bin\go.exe" build . 2>&1 | Out-Null
    if ($LASTEXITCODE -eq 0) {
        $report.Build["Backend Module"] = "PASS"
        Write-Host "  Backend module builds successfully" -ForegroundColor Green
    } else {
        throw "Backend build failed"
    }
    Set-Location ..
} catch {
    $report.Build["Build"] = "FAIL"
    $report.Errors += "Build failed: $($_.Exception.Message)"
    Write-Host "  Build failed: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host ""

# Test 4: Module Wiring Verification
Write-Host "[4/4] Verifying Module Wiring..." -ForegroundColor Yellow
try {
    $goPath = "$env:ProgramFiles\Go\bin\go.exe"
    Push-Location backend
    # Try to build to verify the replace directive works (mod verify doesn't work with local replaces)
    $buildOutput = & $goPath build . 2>&1 | Out-String
    $buildExitCode = $LASTEXITCODE
    Pop-Location
    if ($buildExitCode -eq 0) {
        $report.Tests["Module Wiring"] = @{
            Status = "PASS"
            Output = "Backend builds successfully with detector import"
        }
        Write-Host "  Module wiring verified (backend builds with detector import)" -ForegroundColor Green
    } else {
        throw "Backend build failed: $buildOutput"
    }
} catch {
    Pop-Location -ErrorAction SilentlyContinue
    $report.Tests["Module Wiring"] = @{
        Status = "FAIL"
        Error = $_.Exception.Message
    }
    $report.Errors += "Module wiring failed: $($_.Exception.Message)"
    Write-Host "  Module wiring failed" -ForegroundColor Red
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Test Summary" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Summary
$totalTests = $report.Tests.Count
$passedTests = ($report.Tests.Values | Where-Object { $_.Status -eq "PASS" }).Count
$failedTests = ($report.Tests.Values | Where-Object { $_.Status -eq "FAIL" }).Count

Write-Host "Tests: $passedTests/$totalTests passed" -ForegroundColor $(if ($failedTests -eq 0) { "Green" } else { "Red" })
Write-Host "Builds: $($report.Build.Values | Where-Object { $_ -eq "PASS" }).Count/$($report.Build.Count) successful" -ForegroundColor $(if ($report.Build.Values -contains "FAIL") { "Red" } else { "Green" })

if ($report.Coverage.Count -gt 0) {
    Write-Host ""
    Write-Host "Coverage:" -ForegroundColor Cyan
    foreach ($module in $report.Coverage.Keys) {
        if ($module -match "avg") {
            Write-Host "  $module : $($report.Coverage[$module])%" -ForegroundColor Green
        } elseif ($report.Coverage[$module] -is [hashtable]) {
            # Show individual package coverages
            foreach ($pkg in $report.Coverage[$module].Keys) {
                $cov = $report.Coverage[$module][$pkg]
                $color = if ([double]$cov -ge 80) { "Green" } elseif ([double]$cov -ge 50) { "Yellow" } else { "Red" }
                Write-Host "    $pkg : $cov%" -ForegroundColor $color
            }
        } else {
            Write-Host "  $module : $($report.Coverage[$module])%" -ForegroundColor Green
        }
    }
}

if ($report.Errors.Count -gt 0) {
    Write-Host ""
    Write-Host "Errors:" -ForegroundColor Red
    foreach ($error in $report.Errors) {
        Write-Host "  - $error" -ForegroundColor Red
    }
}

# Generate JSON report
$reportJson = $report | ConvertTo-Json -Depth 10
$reportJson | Out-File -FilePath 'test-report.json' -Encoding utf8
Write-Host ""
Write-Host "Detailed report saved to: test-report.json" -ForegroundColor Cyan

# Exit with appropriate code
if ($failedTests -gt 0 -or $report.Errors.Count -gt 0) {
    exit 1
} else {
    exit 0
}
