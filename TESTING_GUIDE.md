# Pasteguard CLI & HTTP Server - Complete Manual Testing Guide

## Prerequisites
- Go installed and in PATH
- Terminal/Command Prompt access
- PowerShell (for Windows)
- curl or similar HTTP client (for HTTP server testing)

## Step 1: Build the Binary

```powershell
# Build the executable
go build -o pasteguard.exe .

# Verify it was created
dir pasteguard.exe
```

**Expected**: `pasteguard.exe` file should be created

## Step 2: Test Basic Functionality

### Test with --text flag (regular text)
```powershell
.\pasteguard.exe --text "This is just regular text"
```

**Expected Output**:
```json
{
  "overall_risk": "low",
  "risk_rationale": "No issues detected",
  "findings": []
}
```

### Test with empty string
```powershell
.\pasteguard.exe --text ""
```

**Expected Output**:
```json
{
  "overall_risk": "low",
  "risk_rationale": "No issues detected",
  "findings": []
}
```

### Test with stdin
```powershell
echo "Regular text input" | .\pasteguard.exe
```

**Expected**: Same as above - low risk, no findings

## Step 3: Test PEM Private Key Detection

### Test RSA Private Key
```powershell
.\pasteguard.exe --text "-----BEGIN RSA PRIVATE KEY-----`nMIIEpAIBAAKCAQEAyK8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8v`n-----END RSA PRIVATE KEY-----"
```

**Expected Output**:
- `overall_risk: "high"`
- Finding with `type: "pem_private_key"`
- `severity: "high"`
- `confidence: "high"`
- Redacted reason (should show `----...----` - majority masked)
- `line_number: 1`

### Test EC Private Key
```powershell
.\pasteguard.exe --text "-----BEGIN EC PRIVATE KEY-----`nMHcCAQEEIAKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKj`n-----END EC PRIVATE KEY-----"
```

**Expected**: Similar to RSA key detection

### Test Generic Private Key
```powershell
.\pasteguard.exe --text "-----BEGIN PRIVATE KEY-----`nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQC7VJTUt9Us8cKj`n-----END PRIVATE KEY-----"
```

**Expected**: Should detect as PEM private key

## Step 4: Test JWT Detection

### Test JWT Token
```powershell
.\pasteguard.exe --text 'token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"'
```

**Expected Output**:
- `overall_risk: "high"`
- Finding with `type: "jwt_token"`
- `severity: "high"`
- `confidence: "high"`
- Redacted JWT (should show `eyJh...sw5c` - first 4 and last 4 chars)
- `line_number: 1`

### Test Multiple JWTs
```powershell
.\pasteguard.exe --text 'token1 = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"`ntoken2 = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiI5ODc2NTQzMjEwIn0.abc123def456ghi789jkl012mno345pqr678stu901vwx234"'
```

**Expected**: Should detect both JWTs, multiple findings

## Step 5: Test Password Assignment Detection

### Test Password Assignment (with quotes)
```powershell
.\pasteguard.exe --text 'password = "mySecretPassword123"'
```

**Expected Output**:
- `overall_risk: "high"`
- Finding with `type: "password_assignment"`
- `severity: "high"`
- `confidence: "medium"`
- Redacted password (should show `mySe...d123` - first 4 and last 4 chars)
- `line_number: 1`
- **All metrics verified**: type, severity, confidence, reason (redacted), line_number

### Test Password Assignment (without quotes - PowerShell behavior)
```powershell
.\pasteguard.exe --text 'password = secret123'
```

**Expected**: Should still detect password even when PowerShell strips quotes
- `overall_risk: "high"`
- Finding with `type: "password_assignment"`
- All metrics present and correct

### Test API Key
```powershell
.\pasteguard.exe --text 'api_key = "sk-1234567890abcdefghijklmnop"'
```

**Expected**: Similar detection with `type: "password_assignment"`
- All metrics: type, severity, confidence, reason, line_number

### Test Secret
```powershell
.\pasteguard.exe --text 'secret = "super_secret_value_12345"'
```

**Expected**: Similar detection with all metrics

### Test Passwd
```powershell
.\pasteguard.exe --text 'passwd = "mypassword"'
```

**Expected**: Should detect with all metrics

### Test Colon Syntax
```powershell
.\pasteguard.exe --text 'password: "secret123"'
```

**Expected**: Should detect with all metrics

### Verify Password Detection Metrics
```powershell
# Test and verify all fields are present
.\pasteguard.exe --text 'password = "secret123"' | ConvertFrom-Json | Select-Object -ExpandProperty findings | Format-List
```

**Expected**: Each finding should have:
- `type`: "password_assignment"
- `severity`: "high"
- `confidence`: "medium"
- `reason`: redacted string (contains "...")
- `line_number`: number > 0

## Step 6: Test Token Heuristics Rule

### Test High-Entropy Token Near Keyword (HIGH severity)
```powershell
.\pasteguard.exe --text 'token = "AbCdEfGhIjKlMnOpQrStUvWxYz1234567890+/=AbCdEfGhIjKlMnOpQrStUvWx"'
```

**Expected Output**:
- `overall_risk: "high"` (if any HIGH finding)
- Finding with `type: "token_heuristics"`
- `severity: "high"` (long token + near keyword = high score)
- `confidence: "high"` (near keyword with high score)
- Redacted token (should show `AbCd...UvWx` - majority masked, first 4 and last 4)
- `line_number: 1`

### Test Base64-like Token
```powershell
.\pasteguard.exe --text 'api_key = "SGVsbG9Xb3JsZFRoaXNJc0Jhc2U2NA=="'
```

**Expected**: Should detect as `token_heuristics`

### Test Hex-like Token
```powershell
.\pasteguard.exe --text 'secret = "deadbeef1234567890abcdef1234567890abcdef"'
```

**Expected**: Should detect as `token_heuristics`

### Test URL-Safe Token
```powershell
.\pasteguard.exe --text 'access_token = "AbCdEfGhIjKlMnOpQrStUvWxYz1234567890_-"'
```

**Expected**: Should detect as `token_heuristics`

### Test Medium Severity Token
```powershell
.\pasteguard.exe --text 'some_var = "AbCdEfGhIjKlMnOpQrStUvWx"'
```

**Expected**: May detect with `severity: "medium"` if score is 3-4

## Step 7: Verify Conservative Detection (Should NOT Detect)

### Test UUID (Should be ignored)
```powershell
.\pasteguard.exe --text 'uuid = "550e8400-e29b-41d4-a716-446655440000"'
```

**Expected Output**:
```json
{
  "overall_risk": "low",
  "risk_rationale": "No issues detected",
  "findings": []
}
```

### Test Hash (Should be ignored)
```powershell
.\pasteguard.exe --text 'sha256_hash = "a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3"'
```

**Expected**: `overall_risk: "low"`, no findings

### Test MD5 Checksum (Should be ignored)
```powershell
.\pasteguard.exe --text 'md5_checksum = "5d41402abc4b2a76b9719d911017c592"'
```

**Expected**: `overall_risk: "low"`, no findings

### Test Commit Hash (Should be ignored)
```powershell
.\pasteguard.exe --text 'commit = "a1b2c3d4e5f6789012345678901234567890abcd"'
```

**Expected**: `overall_risk: "low"`, no findings

### Test Version Number (Should be ignored or low severity)
```powershell
.\pasteguard.exe --text 'version = "1.2.3.4.5.6.7.8.9.10.11.12.13.14.15.16.17.18.19.20"'
```

**Expected**: Should not detect or very low severity

## Step 8: Test Redaction (Verify No Secrets Leak)

### Test that full secrets are NOT in output
```powershell
# Run with a secret
.\pasteguard.exe --text 'password = "myVeryLongSecretPassword12345"' | Select-String -Pattern "myVeryLongSecretPassword12345"
```

**Expected**: No output (empty) - the full password should NOT appear

### Test that redacted version IS in output
```powershell
.\pasteguard.exe --text 'password = "myVeryLongSecretPassword12345"' | Select-String -Pattern "..."
```

**Expected**: Should find the redacted version with `...`

### Test JWT Redaction
```powershell
.\pasteguard.exe --text 'token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"' | Select-String -Pattern "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"
```

**Expected**: No output - full JWT should NOT appear

### Test Token Heuristics Redaction (Majority Masked)
```powershell
.\pasteguard.exe --text 'access_token = "AbCdEfGhIjKlMnOpQrStUvWxYz1234567890+/=AbCdEfGhIjKlMnOp"'
```

**Expected**: 
- Should show `AbCd...MnOp` (first 4 and last 4)
- Majority of token should be masked (>50%)
- Full token should NOT appear in JSON

## Step 9: Test Multiple Findings

### Test Multiple Secrets
```powershell
.\pasteguard.exe --text "password = `"secret123`"`napi_key = `"sk-1234567890`"`ntoken = `"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U`""
```

**Expected Output**:
- `overall_risk: "high"` (any HIGH finding = overall HIGH)
- Multiple findings in the array
- All secrets properly redacted
- Each finding has: `type`, `severity`, `confidence`, `reason`, `line_number`

### Test Mixed Severities
```powershell
.\pasteguard.exe --text "password = `"secret123`"`nsome_var = `"AbCdEfGhIjKlMnOpQrSt`""
```

**Expected**: Should show both findings with appropriate severities

## Step 9.5: Test Overlap Merging

### Test Overlapping Findings (Password + Token Heuristics)
When multiple rules detect the same secret, they should be merged into a single finding.

```powershell
.\pasteguard.exe --text 'api_key = "AbCdEfGhIjKlMnOpQrStUvWxYz1234567890abcdef"'
```

**Expected Output**:
- `overall_risk: "high"`
- **Only 1 finding** (password_assignment and token_heuristics merged)
- `severity: "high"` (highest from merged findings)
- `confidence: "high"` (maximum from merged findings - token_heuristics has "high")
- `reason`: redacted, concatenated from both rules
- `line_number: 1`

**Verification**: The finding should have `confidence: "high"` because token_heuristics has "high" confidence, even though password_assignment has "medium" confidence.

### Test Overlapping Findings (Same Secret, Different Rules)
```powershell
.\pasteguard.exe --text 'password = "AbCdEfGhIjKlMnOpQrStUvWxYz1234567890+/=AbCdEfGhIjKlMnOp"'
```

**Expected**: 
- Should merge password_assignment and token_heuristics if they overlap
- Merged finding should have highest severity and max confidence
- Reason should be concatenated from both detections

### Test Non-Overlapping Findings (Should NOT Merge)
```powershell
.\pasteguard.exe --text "password = `"secret123`"`napi_key = `"secret123`""
```

**Expected Output**:
- `overall_risk: "high"`
- **2 separate findings** (different line numbers, no overlap)
- Each finding on its own line
- Findings are sorted by line number

### Test Adjacent Findings (Should Merge if Overlapping)
```powershell
.\pasteguard.exe --text 'token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"'
```

**Expected**: If JWT and token_heuristics both detect the same token and their byte ranges overlap, they should merge.

### Verify Deterministic Sorting
```powershell
# Run the same command multiple times
.\pasteguard.exe --text "password = `"secret1`"`napi_key = `"secret2`"`ntoken = `"secret3`""
.\pasteguard.exe --text "password = `"secret1`"`napi_key = `"secret2`"`ntoken = `"secret3`""
.\pasteguard.exe --text "password = `"secret1`"`napi_key = `"secret2`"`ntoken = `"secret3`""
```

**Expected**: All three runs should produce **identical JSON output** (same order of findings, same structure)

### Verify Merged Finding Properties
```powershell
.\pasteguard.exe --text 'api_key = "AbCdEfGhIjKlMnOpQrStUvWxYz1234567890abcdef"' | ConvertFrom-Json | Select-Object -ExpandProperty findings | Format-List
```

**Expected**: If findings are merged:
- Only 1 finding in the array
- `severity` is "high" (highest from merged findings)
- `confidence` is "high" (maximum from merged findings)
- `reason` contains concatenated information from both rules
- `line_number` is the minimum line number from merged findings

## Step 10: Test Edge Cases

### Test Empty Input
```powershell
.\pasteguard.exe --text ""
```

**Expected**: `overall_risk: "low"`, empty findings

### Test Comments (Should ignore)
```powershell
.\pasteguard.exe --text "// password = `"commented out`""
```

**Expected**: Should not detect commented passwords

### Test Already Redacted (Should ignore)
```powershell
.\pasteguard.exe --text 'password = "abcd...xyz"'
```

**Expected**: Should not detect already redacted values

### Test Low Entropy (Should ignore)
```powershell
.\pasteguard.exe --text 'token = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"'
```

**Expected**: Should not detect low-entropy strings

## Step 11: Test with Files (Stdin)

### Create a test file
```powershell
# Create test file with secrets
@"
password = "test123"
api_key = "sk-test123456"
token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"
"@ | Out-File -FilePath test_secrets.txt -Encoding utf8

# Test with file input
Get-Content test_secrets.txt | .\pasteguard.exe
```

**Expected**: Should detect all secrets from the file

### Test with PEM key file
```powershell
@"
-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAyK8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8v
Z8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ
-----END RSA PRIVATE KEY-----
"@ | Out-File -FilePath test_key.txt -Encoding utf8

Get-Content test_key.txt | .\pasteguard.exe
```

**Expected**: Should detect PEM key

## Step 12: Verify JSON Structure

### Check JSON is valid
```powershell
.\pasteguard.exe --text "test" | ConvertFrom-Json | ConvertTo-Json -Depth 10
```

**Expected**: Should output valid JSON with:
- `overall_risk` field (string: "low", "medium", or "high")
- `risk_rationale` field (string)
- `findings` array
- Each finding has:
  - `type` (string)
  - `severity` (string: "high", "medium", or "low")
  - `confidence` (string: "high", "medium", or "low")
  - `reason` (string - redacted)
  - `line_number` (number)

### Verify no RawMatch field
```powershell
.\pasteguard.exe --text 'password = "secret123"' | Select-String -Pattern "RawMatch"
```

**Expected**: No output - `RawMatch` should never appear in JSON

## Step 13: Test Risk Scoring

### Test HIGH Risk (any HIGH finding)
```powershell
.\pasteguard.exe --text 'password = "secret123"'
```

**Expected**: `overall_risk: "high"`, `risk_rationale: "High severity issues detected"`

### Test LOW Risk (no findings)
```powershell
.\pasteguard.exe --text "Regular text"
```

**Expected**: `overall_risk: "low"`, `risk_rationale: "No issues detected"`

### Test Multiple HIGH Findings
```powershell
.\pasteguard.exe --text "password = `"secret`"`ntoken = `"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U`""
```

**Expected**: `overall_risk: "high"` (any HIGH = overall HIGH)

## Step 14: Verify All Rules Are Active

### Test that all 4 rules are working
```powershell
# PEM
.\pasteguard.exe --text "-----BEGIN RSA PRIVATE KEY-----`nMIIEpAIBAAKCAQEAyK8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8v`n-----END RSA PRIVATE KEY-----"

# JWT
.\pasteguard.exe --text 'token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"'

# Password
.\pasteguard.exe --text 'password = "secret123"'

# Token Heuristics
.\pasteguard.exe --text 'api_key = "AbCdEfGhIjKlMnOpQrStUvWxYz1234567890"'
```

**Expected**: Each should detect the appropriate type

## Step 15: HTTP Server Mode

### Starting the Server

```powershell
# Start server on default port 8787
.\pasteguard.exe serve

# Start server on custom port
.\pasteguard.exe serve --addr :8080

# Start server on specific host and port
.\pasteguard.exe serve --addr localhost:8787
```

**Expected**: Server starts and logs "Starting pasteguard server on [address]"

### Testing Health Endpoint

```powershell
# Test health endpoint
curl http://localhost:8787/health
```

**Expected Output**:
```json
{
  "status": "ok"
}
```

### Testing Analyze Endpoint

#### Basic Analysis Request
```powershell
# Using curl
curl -X POST http://localhost:8787/analyze `
  -H "Content-Type: application/json" `
  -d '{\"text\": \"password = \\\"secret123\\\"\"}'
```

**Expected Output**:
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

#### Using PowerShell Invoke-RestMethod
```powershell
$body = @{
    text = "password = `"secret123`""
} | ConvertTo-Json

Invoke-RestMethod -Uri http://localhost:8787/analyze `
  -Method POST `
  -ContentType "application/json" `
  -Body $body
```

**Expected**: Same JSON response as above

#### Test PEM Key Detection
```powershell
$body = @{
    text = "-----BEGIN RSA PRIVATE KEY-----`nMIIEpAIBAAKCAQEAyK8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8v`n-----END RSA PRIVATE KEY-----"
} | ConvertTo-Json

Invoke-RestMethod -Uri http://localhost:8787/analyze `
  -Method POST `
  -ContentType "application/json" `
  -Body $body
```

**Expected**: Should detect PEM key with `overall_risk: "high"`

#### Test JWT Detection
```powershell
$body = @{
    text = "token = `"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U`""
} | ConvertTo-Json

Invoke-RestMethod -Uri http://localhost:8787/analyze `
  -Method POST `
  -ContentType "application/json" `
  -Body $body
```

**Expected**: Should detect JWT with `overall_risk: "high"`

#### Test Empty Text
```powershell
$body = @{
    text = ""
} | ConvertTo-Json

Invoke-RestMethod -Uri http://localhost:8787/analyze `
  -Method POST `
  -ContentType "application/json" `
  -Body $body
```

**Expected**: Should return `overall_risk: "low"` with empty findings

### Testing Request Size Limits

```powershell
# Create a large text (over 1MB)
$largeText = "a" * (2 * 1024 * 1024)  # 2MB
$body = @{
    text = $largeText
} | ConvertTo-Json

try {
    Invoke-RestMethod -Uri http://localhost:8787/analyze `
      -Method POST `
      -ContentType "application/json" `
      -Body $body
} catch {
    Write-Host "Expected error: $_"
}
```

**Expected**: Should return HTTP 400 Bad Request with error message

### Testing Rate Limiting

```powershell
# Make 100 requests (the limit)
1..100 | ForEach-Object {
    $body = @{
        text = "test $_"
    } | ConvertTo-Json
    
    Invoke-RestMethod -Uri http://localhost:8787/analyze `
      -Method POST `
      -ContentType "application/json" `
      -Body $body | Out-Null
}

# 101st request should be rate limited
$body = @{
    text = "test 101"
} | ConvertTo-Json

try {
    Invoke-RestMethod -Uri http://localhost:8787/analyze `
      -Method POST `
      -ContentType "application/json" `
      -Body $body
} catch {
    # Should get 429 Too Many Requests
    Write-Host "Rate limit error (expected): $_"
}
```

**Expected**: 
- First 100 requests should succeed (HTTP 200)
- 101st request should return HTTP 429 with `{"error": "Rate limit exceeded"}`

### Testing Invalid Requests

#### Test Invalid JSON
```powershell
curl -X POST http://localhost:8787/analyze `
  -H "Content-Type: application/json" `
  -d "invalid json"
```

**Expected**: HTTP 400 Bad Request with `{"error": "Invalid JSON"}`

#### Test Wrong HTTP Method
```powershell
curl http://localhost:8787/analyze
```

**Expected**: HTTP 405 Method Not Allowed with `{"error": "Method not allowed"}`

#### Test Missing Text Field
```powershell
curl -X POST http://localhost:8787/analyze `
  -H "Content-Type: application/json" `
  -d "{}"
```

**Expected**: Should still process (empty text = low risk)

### Security Verification

#### Verify No Input Logging
1. Start the server: `.\pasteguard.exe serve`
2. Send a request with sensitive data:
   ```powershell
   $body = @{
       text = "password = `"super_secret_password_12345`""
   } | ConvertTo-Json
   
   Invoke-RestMethod -Uri http://localhost:8787/analyze `
     -Method POST `
     -ContentType "application/json" `
     -Body $body
   ```
3. Check server logs/console output

**Expected**: 
- Server logs should NOT contain the password text
- Only server start messages and error messages (without user input) should appear
- Response should contain redacted version only

#### Verify Redaction in HTTP Response
```powershell
$body = @{
    text = "api_key = `"AbCdEfGhIjKlMnOpQrStUvWxYz1234567890abcdef`""
} | ConvertTo-Json

$response = Invoke-RestMethod -Uri http://localhost:8787/analyze `
  -Method POST `
  -ContentType "application/json" `
  -Body $body

# Check that full secret is not in response
$responseJson = $response | ConvertTo-Json -Depth 10
if ($responseJson -match "AbCdEfGhIjKlMnOpQrStUvWxYz1234567890abcdef") {
    Write-Host "ERROR: Full secret found in response!"
} else {
    Write-Host "OK: Secret is properly redacted"
}
```

**Expected**: Full secret should NOT appear in response, only redacted version

### Testing CLI Mode Still Works

After implementing HTTP server, verify CLI mode still functions:

```powershell
# Test CLI mode
.\pasteguard.exe --text "password = `"secret123`""

# Test stdin
echo "password = `"secret123`"" | .\pasteguard.exe
```

**Expected**: CLI mode should work exactly as before, producing same JSON output

## Quick Verification Checklist

### Basic Functionality
- [ ] Binary builds successfully
- [ ] Regular text returns low risk
- [ ] Empty string works (`--text ""`)
- [ ] Exit code is always 0

### Detection Rules
- [ ] PEM keys are detected and redacted
- [ ] JWTs are detected and redacted
- [ ] Passwords are detected and redacted (with and without quotes)
- [ ] Token heuristics detects high-entropy tokens
- [ ] Token heuristics returns HIGH severity for high-scoring tokens near keywords
- [ ] Token heuristics returns MEDIUM severity for lower-scoring tokens
- [ ] All 4 rules (PEM, JWT, Password, TokenHeuristics) are active

### Conservative Detection
- [ ] UUIDs are NOT detected (conservative)
- [ ] Hashes are NOT detected (conservative)
- [ ] Commit hashes are NOT detected (conservative)
- [ ] Version numbers are NOT detected (conservative)

### Redaction & Security
- [ ] Full secrets never appear in output
- [ ] Redacted versions appear with `...`
- [ ] Token heuristics masks majority of token (>50%)
- [ ] RawMatch field never appears in JSON

### Metrics & Structure
- [ ] All findings have all required fields: type, severity, confidence, reason, line_number
- [ ] JSON structure is valid
- [ ] Severity values are valid (high/medium/low)
- [ ] Confidence values are valid (high/medium/low)
- [ ] Line numbers are accurate

### Risk Scoring
- [ ] Risk scoring: HIGH if any HIGH finding
- [ ] Multiple findings work correctly
- [ ] Mixed severities handled correctly

### Overlap Merging
- [ ] Overlapping findings are merged into one
- [ ] Merged findings have highest severity from all merged findings
- [ ] Merged findings have maximum confidence from all merged findings
- [ ] Merged findings have concatenated reasons from all merged findings
- [ ] Non-overlapping findings remain separate
- [ ] Findings are sorted deterministically (same input = same output order)
- [ ] Multiple runs produce identical JSON output

### Overlap Merging & Sorting
- [ ] Overlapping findings merge correctly
- [ ] Merged findings preserve highest severity
- [ ] Merged findings preserve maximum confidence
- [ ] Reasons are concatenated in merged findings
- [ ] Findings are sorted deterministically
- [ ] Multiple runs produce identical output

### HTTP Server
- [ ] Server starts successfully with `serve` command
- [ ] Health endpoint returns `{"status": "ok"}`
- [ ] POST /analyze accepts valid JSON with `text` field
- [ ] POST /analyze returns correct analysis results
- [ ] PEM keys are detected via HTTP API
- [ ] JWTs are detected via HTTP API
- [ ] Passwords are detected via HTTP API
- [ ] Token heuristics work via HTTP API
- [ ] Empty text returns low risk
- [ ] Request size limit (1MB) is enforced
- [ ] Rate limiting works (100 requests per minute per IP)
- [ ] Invalid JSON returns 400 error
- [ ] Wrong HTTP method returns 405 error
- [ ] No user input is logged to console
- [ ] Responses contain redacted secrets only
- [ ] CLI mode still works after server implementation

### Test Coverage
- [ ] All unit tests pass (95+ tests)
- [ ] CLI tests pass (13 tests)
- [ ] HTTP server tests pass (15 tests)
- [ ] Redaction tests pass (8 tests)
- [ ] All rule tests pass (50+ tests)
- [ ] Merge and sort tests pass (11 tests)

## Running Unit Tests

```powershell
# Run all tests (including new CLI tests)
go test -v ./...

# Run specific test package
go test -v ./detector

# Run CLI tests only
go test -v ./... -run TestCLI

# Run password detection tests
go test -v ./... -run TestCLIPasswordDetection
go test -v ./detector -run TestPasswordRule

# Run all rule tests
go test -v ./detector -run TestTokenHeuristics
go test -v ./detector -run TestRedact
go test -v ./detector -run TestPEMRule
go test -v ./detector -run TestJWTRule

# Run with coverage
go test ./... -cover

# Run specific test suites
go test -v ./... -run TestCLIRedactionNoSecretsLeak
go test -v ./... -run TestCLIRiskScoring
go test -v ./... -run TestCLIAllFindingMetrics

# Run overlap merging tests
go test -v ./detector -run TestMerge
go test -v ./detector -run TestSort
```

**Expected**: All tests should pass (95+ tests total)

# Run HTTP server tests
go test -v ./server

# Run all tests including server
go test -v ./...

### Test Coverage Summary
- **CLI Tests**: 13 tests (including password, PEM, JWT, token heuristics, redaction, risk scoring)
- **HTTP Server Tests**: 15 tests (health, analyze endpoint, rate limiting, size limits, security)
- **Rule Tests**: 50+ tests (PEM, JWT, Password, TokenHeuristics)
- **Redaction Tests**: 8 tests (verify no secrets leak)
- **Engine Tests**: 10+ tests (risk scoring, engine functionality)
- **Merge & Sort Tests**: 11 tests (overlap merging, deterministic sorting)
- **Total**: 95+ tests, all passing

## Troubleshooting

### If Go is not found:
```powershell
# Add Go to PATH (adjust path as needed)
$env:Path += ";C:\Program Files\Go\bin"
```

### If JSON output is hard to read:
```powershell
# Pretty print JSON
.\pasteguard.exe --text "test" | ConvertFrom-Json | ConvertTo-Json -Depth 10
```

### To save output to file:
```powershell
.\pasteguard.exe --text "test" | Out-File -FilePath output.json -Encoding utf8
```

### To verify redaction is working:
```powershell
# This should return empty (no full secret)
.\pasteguard.exe --text 'password = "mySecret123"' | Select-String -Pattern "mySecret123"

# This should find the redacted version
.\pasteguard.exe --text 'password = "mySecret123"' | Select-String -Pattern "..."
```

## Expected Test Results Summary

| Test Case | Expected Risk | Expected Findings |
|-----------|--------------|-------------------|
| Regular text | LOW | 0 |
| Empty string | LOW | 0 |
| PEM key | HIGH | 1 (pem_private_key) |
| JWT token | HIGH | 1 (jwt_token) |
| Password | HIGH | 1 (password_assignment) |
| Token near keyword (long) | HIGH | 1+ (token_heuristics, HIGH) |
| Token without keyword | MEDIUM/LOW | 0-1 (token_heuristics, MEDIUM if detected) |
| UUID | LOW | 0 |
| Hash in safe context | LOW | 0 |
| Multiple secrets | HIGH | 3+ findings |
| Commented password | LOW | 0 |
| Already redacted | LOW | 0 |

## Notes

### Security
- All secrets should be redacted (majority masked for token_heuristics)
- Full secrets should NEVER appear in JSON output
- RawMatch field is never included in JSON output
- Redaction ensures >50% of token_heuristics tokens are masked

### Detection Behavior
- Token heuristics is conservative - ignores safe contexts (UUID, hash, commit, etc.)
- Password detection works with or without quotes (handles PowerShell quote stripping)
- Overall risk is HIGH if ANY finding has HIGH severity
- Multiple rules can detect different secrets in the same input

### Metrics
- All findings include: type, severity, confidence, reason (redacted), line_number
- Severity: "high", "medium", or "low"
- Confidence: "high", "medium", or "low"
- Reason: always redacted (contains "..." for secrets)

### Testing
- Exit code is always 0 (success)
- 80+ unit tests cover all functionality
- CLI tests verify end-to-end behavior
- Merge and sort tests verify overlap handling
- All tests should pass before deployment

### Overlap Merging Behavior
- Findings with overlapping byte ranges are automatically merged
- Merged findings take the highest severity from all merged findings
- Merged findings take the maximum confidence from all merged findings
- Reasons from all merged findings are concatenated with ", "
- Line number is the minimum line number from merged findings
- Findings are sorted deterministically by line number, then byte start position
- Same input always produces identical output (deterministic)
