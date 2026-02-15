# E2E Test Setup Steps

## Quick Start

1. **Fix Go dependencies** (required first time):
   ```powershell
   # From project root (C:\Users\anand\Development\hacknc)
   go mod tidy
   ```

2. **Install E2E dependencies**:
   ```powershell
   cd e2e
   npm install
   npm run install:browsers
   ```

3. **Run tests**:
   ```powershell
   npm test
   ```

## Detailed Steps

### Step 1: Fix Go Dependencies

The root `go.mod` needs `go.sum` entries. Run from project root:

```powershell
cd C:\Users\anand\Development\hacknc
go mod tidy
```

This will:
- Download missing dependencies
- Generate `go.sum` file
- Resolve all module dependencies

### Step 2: Install Node Dependencies

```powershell
cd e2e
npm install
```

This installs:
- `@playwright/test` - Testing framework
- `express` - Test app server

### Step 3: Install Playwright Browsers

```powershell
npm run install:browsers
```

This downloads Chromium browser for Playwright.

### Step 4: Run Tests

```powershell
npm test
```

The test suite automatically:
1. Starts backend server (`BACKEND_TEST_MODE=1`) on port 8080
2. Starts test app server on port 4173
3. Launches Chromium with extension loaded
4. Sets test overrides (malicious domains)
5. Runs all E2E tests

## Troubleshooting

### Error: "missing go.sum entry"
**Solution**: Run `go mod tidy` from project root

### Error: "Port 8080 already in use"
**Solution**: 
```powershell
# Find process
netstat -ano | findstr :8080
# Kill it (replace <PID> with actual number)
taskkill /F /PID <PID>
```

### Error: "Timed out waiting for http://localhost:8080/health"
**Possible causes**:
- Backend failed to start (check go.sum issue above)
- Port 8080 blocked by firewall
- Another process using port 8080

**Solution**: 
1. Fix go.sum: `go mod tidy`
2. Kill any process on port 8080
3. Try again: `npm test`

## Manual Backend Testing

To test backend manually:

```powershell
# From project root
$env:BACKEND_TEST_MODE="1"
go run main.go serve --addr :8080
```

Then in another terminal:
```powershell
# Test health endpoint
curl http://localhost:8080/health
# Should return: {"ok":true}
```

## Environment Variables

- `BACKEND_TEST_MODE=1` - Enables test mode (in-memory storage, no AWS)
- `TESTAPP_PORT=4173` - Test app server port (default: 4173)
- `HEADLESS=1` - Run Playwright in headless mode (default: headed)

