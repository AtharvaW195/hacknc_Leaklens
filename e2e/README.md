# PasteGuard E2E Tests

End-to-end tests for the PasteGuard browser extension using Playwright.

## Prerequisites

- Node.js 16+ installed
- Go 1.21+ installed and in PATH
- Chrome/Chromium browser (installed via `npm run install:browsers`)

## Setup

1. **Fix Go dependencies** (first time only):
```bash
# From project root directory
go mod tidy
```

2. **Install E2E dependencies**:
```bash
cd e2e
npm install
npm run install:browsers
```

3. **Run tests** (backend and testapp are started automatically):
```bash
npm test
```

The test suite will:
- Start the backend server in test mode (`BACKEND_TEST_MODE=1`)
- Start the test app server on port 4173
- Launch Chromium with the extension loaded
- Set deterministic test overrides (malicious domains list)
- Run all E2E tests

## Troubleshooting

### "missing go.sum entry" error
If you see this error, run from project root:
```bash
go mod tidy
```

### Port 8080 already in use
Kill any existing backend process:
```powershell
# Find process using port 8080
netstat -ano | findstr :8080
# Kill it (replace PID with actual process ID)
taskkill /F /PID <PID>
```

### Port 4173 already in use
Kill any existing testapp process or change `TESTAPP_PORT` in `global-setup.js`

## Manual Testing

To test the backend manually:
```bash
# In project root
cd backend
BACKEND_TEST_MODE=1 go run .
# Or from root:
BACKEND_TEST_MODE=1 go run main.go serve --addr :8080
```

The backend will:
- Use in-memory storage (no AWS/S3 required)
- Enforce one-time view links (first view: 200, second view: 410 with "LINK_USED")
- Provide `/health` endpoint returning `{"ok": true}`

## Test Structure

- `tests/` - E2E test specifications
- `testapp/` - Test web application with various pages for testing extension features
- `fixtures/` - Test data files (e.g., secret.txt for paste testing)
- `utils/` - Utility functions for extension testing

## Test Pages

The test app (`testapp/`) provides several pages:

- **index.html** - Home page with navigation
- **upload.html** - Tests Secure Share button injection
- **paste.html** - Tests paste guard interception
- **links.html** - Tests link validation and warning icons

## How to Run E2E Tests

### Quick Start

```bash
# From e2e directory
npm test
```

This will:
- Start the backend server in test mode (`BACKEND_TEST_MODE=1`)
- Start the test app server on port 4173
- Launch Chromium with the extension loaded
- Set deterministic test overrides (malicious domains list: `["evil.test"]`)
- Run all E2E tests
- Clean up servers and browser on completion

### Important: Extension Tests Run in Headed Mode

**CRITICAL**: Extension tests (`extension` project) **MUST** run in headed mode (not headless). 
Chromium extensions often fail to load in headless mode, which will cause tests to timeout.

The test suite is configured with two projects:
- **backend** project: Runs backend-only tests headless (faster)
- **extension** project: Runs all extension-related tests with `headless: false` (required for extension loading)

You will see browser windows open during extension tests - this is expected and required.

### Running Specific Test Suites

```bash
# Run only domain blocking tests
npx playwright test blocked-domain

# Run only paste guard tests
npx playwright test paste-guard

# Run only link scanner tests
npx playwright test link-scanner

# Run only secure upload tests
npx playwright test secure-upload

# Run only settings tests
npx playwright test settings

# Run only backend one-time view tests
npx playwright test backend-one-time
```

### Running Tests in Different Modes

```bash
# Run all tests headless (default)
npm test

# Run tests in headed mode (see browser)
npm run test:headed
# Or: npx playwright test --headed

# Run tests in debug mode (step through)
npm run test:debug
# Or: npx playwright test --debug

# Run tests with Playwright UI mode (interactive)
npm run test:ui
# Or: npx playwright test --ui

# Run tests with trace (for debugging failures)
npx playwright test --trace on
```

### Running a Single Test

```bash
# Run a specific test by name
npx playwright test -g "Block evil.test with overlay shown"

# Run tests matching a pattern
npx playwright test blocked-domain
```

## Inspecting Test Traces

When tests fail, Playwright automatically saves traces (videos, screenshots, network logs) for debugging.

### View Traces in Browser

1. Run tests with trace enabled:
   ```bash
   npx playwright test --trace on
   ```

2. After a test fails, open the trace:
   ```bash
   npx playwright show-trace test-results/<test-name>/trace.zip
   ```

3. The trace viewer will open in your browser showing:
   - **Timeline**: Step-by-step execution
   - **Actions**: All user interactions (clicks, typing, etc.)
   - **Network**: All HTTP requests and responses
   - **Console**: JavaScript console logs
   - **Screenshots**: Visual state at each step
   - **Video**: Full test execution video

### Trace Files Location

Traces are saved in `e2e/test-results/`:
- Each test gets its own directory
- Failed tests automatically save traces (configured in `playwright.config.js`)
- Trace files are `.zip` archives containing all debugging data

### Manual Trace Inspection

```bash
# List all trace files
ls test-results/*/trace.zip

# Open a specific trace
npx playwright show-trace test-results/blocked-domain-Block-evil.test/trace.zip
```

### Trace Viewer Features

- **Time Travel**: Click any action to see the page state at that moment
- **Network Inspector**: View all API calls, request/response bodies, headers
- **Console Logs**: See all console.log, console.error from content scripts and background
- **Screenshots**: Visual diff between steps
- **Video**: Watch the full test execution

## Test Structure

### Test Files

The test suite is organized by feature:

- **`blocked-domain.spec.js`** - Malicious domain blocking tests
  - Domain overlay display
  - Subdomain blocking
  - Storage reset between tests
  - E2E mode verification

- **`link-scanner.spec.js`** - Link validation and scanning tests
  - Bad link flagging with warning icons
  - Confirm dialog on click
  - Insecure HTTP link detection
  - Dynamic link scanning

- **`secure-upload.spec.js`** - Secure file upload tests
  - Secure Share button injection
  - Upload modal functionality
  - Progress bar display
  - Error handling

- **`paste-guard.spec.js`** - Paste interception tests
  - Low/high risk paste handling
  - Redacted paste
  - Paste anyway / cancel
  - Convert to Secure Share link
  - Fault tolerance (timeout, errors)

- **`settings.spec.js`** - Extension settings UI tests
  - Settings persistence
  - Toggle paste guard enabled
  - Block threshold configuration
  - Link checker functionality
  - Stats display

- **`backend-one-time.spec.js`** - One-time view link semantics
  - First view succeeds
  - Second view returns 410 with LINK_USED
  - Multiple independent links
  - Invalid link handling

- **`extension.e2e.spec.js`** - Integration tests
  - End-to-end flow verification

### Test Helpers

Located in `e2e/utils/test-helpers.js`:

- **`e2ePaste(page, selector, text)`** - Paste text using E2E message bridge (not clipboard)
- **`resetExtensionState(context, extensionId)`** - Clear storage and set E2E overrides
- **`getExtensionId(context)`** - Get extension ID from service worker
- **`waitForToast(page, substring)`** - Wait for toast notification
- **`waitForContentScript(page)`** - Wait for content script to load

### Test Utilities

Located in `e2e/utils/`:

- **`launch-extension.js`** - Launch browser with extension loaded
- **`prepare-extension.js`** - Prepare extension for testing
- **`wait-http.js`** - Wait for HTTP server to be ready

## Test Best Practices

### Deterministic Tests

- All tests use `data-testid` selectors (no CSS classes that might change)
- Storage is reset between tests via `resetExtensionState()`
- Network mocking is scoped per test with `context.route()`
- No arbitrary sleeps - use `expect().toBeVisible()` and `waitForFunction()`

### Test Independence

- Each test is independent (no shared state)
- Tests can run in parallel (except where `fullyParallel: false` in config)
- Storage is cleared before each test

### Windows Compatibility

- All tests are stable on Windows
- Path handling uses `path.join()` for cross-platform compatibility
- No Unix-specific commands or assumptions

## Requirements

- Node.js 16+
- Go 1.21+ (for backend)
- Chrome/Chromium browser (installed via `npm run install:browsers`)
- Extension built and available in `../extension/`

## Environment Variables

- `BACKEND_TEST_MODE=1` - Set automatically by test setup, enables in-memory storage
- `E2E_EXTENSION_PATH` - Path to extension directory (set automatically)
- `CI` - Set automatically in CI environments
- `HEADLESS=1` - Run browser in headless mode (default for `npm test`)

