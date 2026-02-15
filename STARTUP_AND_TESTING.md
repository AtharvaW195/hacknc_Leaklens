# Startup and Manual Testing Guide

This guide covers starting all servers and testing the complete system, including the new paste interception feature.

## 🚀 Quick Start

### Prerequisites
- Go 1.21+ installed
- Chrome/Edge browser (for extension)
- AWS CLI configured (for backend uploads - optional)
- Terminal/PowerShell access

---

## Step 1: Start the Unified Server

The pasteguard server now includes **all functionality** in a single server:
- `/analyze` - Text analysis for paste interception
- `/api/upload` - File uploads to S3
- `/api/generate-upload-url` - Generate presigned upload URLs
- `/view/<id>` - One-time view links for uploaded files
- `/health` - Health check endpoint

The server defaults to port **8080** (matching the extension configuration).

### Option A: Run from Source (Recommended)

**PowerShell:**
```powershell
# From project root
go run . serve
```

**Unix/Linux/Mac:**
```bash
# From project root
go run . serve
```

### Option B: Build and Run Binary

**PowerShell:**
```powershell
# Build
go build -o pasteguard.exe .

# Run server (defaults to :8080)
.\pasteguard.exe serve
```

**Unix/Linux/Mac:**
```bash
# Build
go build -o pasteguard .

# Run server (defaults to :8080)
./pasteguard serve
```

### Option C: Custom Port

If you need a different port:

**PowerShell:**
```powershell
go run . serve --addr :8787
```

**Unix/Linux/Mac:**
```bash
go run . serve --addr :8787
```

**Expected Output:**
```
Starting pasteguard server on :8080
```

**Keep this terminal window open!** The server must stay running.

### Optional: AWS Configuration (for File Uploads)

If you want to use file upload features, you need AWS credentials:

1. **Set up AWS CLI** (if not already done):
   ```powershell
   aws configure
   ```

2. **Create `.env` file** in project root (optional):
   ```
   AWS_BUCKET_NAME=your-bucket-name
   VIEW_LINK_BASE_URL=http://localhost:8080
   ```

   **Note:** If AWS credentials are not configured, the server will still run but file uploads will return an error. Paste interception will work fine without AWS.

---

## Step 3: Load the Browser Extension

**📖 For detailed extension setup instructions, see [EXTENSION_SETUP.md](EXTENSION_SETUP.md)**

### Quick Steps:

1. **Open Extensions Page:**
   - Chrome: `chrome://extensions/`
   - Edge: `edge://extensions/`

2. **Enable Developer Mode:**
   - Toggle **"Developer mode"** ON (top-right corner)

3. **Load Extension:**
   - Click **"Load unpacked"** button
   - Navigate to: `C:\Users\anand\Development\hacknc\extension`
   - Select the `extension` **folder** (not files inside)

4. **Verify Extension is Active:**
   - Extension appears in list with toggle ON
   - Open any webpage and press **F12** → Console tab
   - Look for: `Privacy Guardrail: Initializing...`
   - Visit a page - you should see a status badge

5. **Reload After Changes:**
   - Click the **🔄 refresh icon** on the extension card
   - Or toggle OFF/ON

**See [EXTENSION_SETUP.md](EXTENSION_SETUP.md) for troubleshooting and detailed management instructions.**

---

## Step 4: Test Paste Interception

### Test 1: Normal Text (Should Paste Normally)

1. Open any webpage (e.g., `https://example.com`)
2. Find an `<input>` or `<textarea>` field
3. Copy some normal text: `"Hello, this is just regular text"`
4. Paste into the field (Ctrl+V / Cmd+V)

**Expected:**
- Text pastes immediately
- No tooltip appears
- No blocking

### Test 2: Secret Text (Should Block)

1. Copy a secret: `password = "mySecretPassword123"`
2. Paste into an input/textarea field

**Expected:**
- Paste is **blocked** (text doesn't appear)
- Red tooltip appears: **"Sensitive paste blocked (HIGH)"**
- Tooltip disappears after 3 seconds

### Test 3: Medium Risk Text

1. Copy text with medium risk secrets
2. Paste into field

**Expected:**
- Paste is blocked
- Tooltip shows: **"Sensitive paste blocked (MEDIUM)"**

### Test 4: Server Offline (Fail Open)

1. **Stop the server** (Ctrl+C in the terminal)
2. Try pasting a secret: `password = "secret123"`

**Expected:**
- Paste **succeeds** (fail open behavior)
- Tooltip appears: **"Analyzer offline"**
- This ensures users aren't blocked if the server is down

### Test 5: Different Input Types

Test paste interception on:
- `<input type="text">` ✅ Should intercept
- `<input type="search">` ✅ Should intercept
- `<input type="email">` ✅ Should intercept
- `<input type="url">` ✅ Should intercept
- `<input type="tel">` ✅ Should intercept
- `<input type="password">` ✅ Should intercept
- `<textarea>` ✅ Should intercept
- `<input type="number">` ❌ Should NOT intercept
- `<input type="checkbox">` ❌ Should NOT intercept

### Test 6: Test on Real Websites

1. Go to a login page (e.g., GitHub, Gmail)
2. Try pasting secrets into password fields
3. Verify blocking works

---

## Step 5: Test Other Extension Features

### Test Malicious Domain Blocking

1. Navigate to a domain in `extension/malicious_domains.json`
2. **Expected:** Red blocking overlay appears

### Test Link Scanning

1. Visit a page with links
2. Links to malicious domains should show warning icons (⚠️)
3. Clicking flagged links shows confirmation dialog

### Test Secure Share Button

1. Find a page with a file input (`<input type="file">`)
2. **Expected:** A "🔒 Secure Share" button appears next to it
3. Click it to test file upload (requires backend server)

---

## Step 6: Manual API Testing

### Test Health Endpoint

**PowerShell:**
```powershell
Invoke-RestMethod -Uri http://localhost:8080/health
```

**Expected:**
```json
{
  "status": "ok"
}
```

### Test Analyze Endpoint Directly

**PowerShell:**
```powershell
$body = @{
    text = "password = `"secret123`""
} | ConvertTo-Json

Invoke-RestMethod -Uri http://localhost:8080/analyze `
  -Method POST `
  -ContentType "application/json" `
  -Body $body
```

**Expected:**
```json
{
  "overall_risk": "high",
  "risk_rationale": "High severity issues detected",
  "findings": [
    {
      "type": "password_assignment",
      "severity": "high",
      "confidence": "medium",
      "reason": "secr...t123",
      "line_number": 1
    }
  ]
}
```

### Test with Different Risk Levels

**Low Risk:**
```powershell
$body = @{ text = "This is just regular text" } | ConvertTo-Json
Invoke-RestMethod -Uri http://localhost:8080/analyze -Method POST -ContentType "application/json" -Body $body
```

**High Risk (JWT):**
```powershell
$body = @{ text = 'token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"' } | ConvertTo-Json
Invoke-RestMethod -Uri http://localhost:8080/analyze -Method POST -ContentType "application/json" -Body $body
```

**High Risk (PEM Key):**
```powershell
$body = @{ text = "-----BEGIN RSA PRIVATE KEY-----`nMIIEpAIBAAKCAQEAyK8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8v`n-----END RSA PRIVATE KEY-----" } | ConvertTo-Json
Invoke-RestMethod -Uri http://localhost:8080/analyze -Method POST -ContentType "application/json" -Body $body
```

---

## Step 7: Test File Upload (Requires AWS Configuration)

### Prerequisites
- Server running on port 8080
- AWS credentials configured (`aws configure`)
- S3 bucket name set in `.env` (or `AWS_BUCKET_NAME` environment variable)

### Test Secure Upload

1. Visit a page with a file input
2. Click "🔒 Secure Share" button
3. Select a file (PDF, DOCX, etc.)
4. **Expected:** File uploads and returns a view-only link

---

## Troubleshooting

### Extension Not Working

1. **Check server is running:**
   ```powershell
   Invoke-RestMethod -Uri http://localhost:8080/health
   ```

2. **Check browser console:**
   - Press F12 → Console tab
   - Look for errors from "Privacy Guardrail"

3. **Reload extension:**
   - Go to `chrome://extensions/`
   - Click refresh icon on extension card

4. **Check manifest permissions:**
   - Verify `host_permissions` includes `http://localhost:8080/*`

### Server Won't Start

1. **Port already in use:**
   ```powershell
   # Check what's using port 8080
   netstat -ano | findstr :8080
   ```

2. **Kill process if needed:**
   ```powershell
   # Find PID from netstat, then:
   taskkill /PID <pid> /F
   ```

### Paste Not Intercepting

1. **Verify input type:**
   - Only `text`, `search`, `email`, `url`, `tel`, `password` inputs and `textarea` are intercepted
   - Other input types (number, checkbox, etc.) are ignored

2. **Check browser console for errors:**
   - Press F12 → Console
   - Look for "Privacy Guardrail" messages

3. **Verify event listener:**
   - The paste listener uses capture phase, so it should catch all pastes
   - Check that `isValidPasteTarget()` returns true for your element

### Analysis Not Working

1. **Test API directly:**
   ```powershell
   $body = @{ text = "test" } | ConvertTo-Json
   Invoke-RestMethod -Uri http://localhost:8080/analyze -Method POST -ContentType "application/json" -Body $body
   ```

2. **Check server logs:**
   - Look at the terminal where the server is running
   - Check for error messages

3. **Verify endpoint:**
   - Extension expects: `http://localhost:8080/analyze`
   - Server provides: `http://localhost:8080/analyze` ✅

---

## Quick Test Checklist

- [ ] Pasteguard server running on port 8080
- [ ] Health endpoint returns `{"status": "ok"}`
- [ ] Analyze endpoint works with test secret
- [ ] Extension loaded in browser
- [ ] Normal text pastes successfully
- [ ] Secret text blocked with tooltip
- [ ] Offline mode fails open (pastes normally)
- [ ] Tooltip appears and disappears correctly
- [ ] Only valid input types are intercepted
- [ ] Other extension features still work (domain blocking, link scanning)

---

## Development Workflow

### Making Changes to Extension

1. Edit files in `extension/` folder
2. Go to `chrome://extensions/`
3. Click refresh icon on extension card
4. Test changes immediately

### Making Changes to Server

1. Stop server (Ctrl+C)
2. Make code changes
3. Restart server
4. Test changes

### Running Multiple Instances

If you need multiple server instances (e.g., for testing):

1. **Terminal 1:** Run server on default port 8080
   ```powershell
   go run . serve
   ```

2. **Terminal 2:** Run server on different port
   ```powershell
   go run . serve --addr :8081
   ```

**Note:** The extension is configured for port 8080, so only the first instance will work with the extension.

---

## Summary

**Complete Setup (All Features):**
1. **Install dependencies** (first time only):
   ```powershell
   go mod tidy
   ```

2. **Start unified server:**
   ```powershell
   go run . serve
   ```
   - Handles `/analyze` (paste interception)
   - Handles `/api/upload` (file uploads)
   - Handles `/view/<id>` (view-only links)
   - Defaults to port 8080 (matches extension)

3. **Load extension in browser**
   - Go to `chrome://extensions/`
   - Enable Developer mode
   - Load unpacked → select `extension` folder

4. **Test all features**
   - Paste interception ✅
   - File uploads ✅ (requires AWS config)
   - Link scanning ✅
   - Domain blocking ✅

**Optional AWS Setup (for File Uploads):**
- Configure AWS CLI: `aws configure`
- Create `.env` file with `AWS_BUCKET_NAME=your-bucket`
- Without AWS, paste interception still works perfectly!

**Testing:**
- Normal paste → Works immediately
- Secret paste → Blocked with tooltip
- Offline → Fails open, shows "Analyzer offline"
- API → Test with PowerShell `Invoke-RestMethod`

---

For more detailed testing instructions, see [TESTING_GUIDE.md](TESTING_GUIDE.md).

