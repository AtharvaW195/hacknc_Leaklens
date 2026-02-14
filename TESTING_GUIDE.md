# Pasteguard CLI - Manual Testing Guide

## Prerequisites
- Go installed and in PATH
- Terminal/Command Prompt access

## Step 1: Build the Binary

```powershell
# Build the executable
go build -o pasteguard.exe .

# Verify it was created
dir pasteguard.exe
```

## Step 2: Test Basic Functionality

### Test with --text flag
```powershell
.\pasteguard.exe --text "This is just regular text"
```

**Expected**: `overall_risk: "low"`, empty findings array

### Test with stdin
```powershell
echo "Regular text input" | .\pasteguard.exe
```

**Expected**: Same as above

## Step 3: Test PEM Private Key Detection

### Test RSA Private Key
```powershell
.\pasteguard.exe --text "-----BEGIN RSA PRIVATE KEY-----`nMIIEpAIBAAKCAQEAyK8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8v`n-----END RSA PRIVATE KEY-----"
```

**Expected**: 
- `overall_risk: "high"`
- Finding with `type: "pem_private_key"`
- `severity: "high"`
- Redacted reason (should show `----...----`)

### Test EC Private Key
```powershell
.\pasteguard.exe --text "-----BEGIN EC PRIVATE KEY-----`nMHcCAQEEIAKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKj`n-----END EC PRIVATE KEY-----"
```

**Expected**: Similar to RSA key detection

## Step 4: Test JWT Detection

### Test JWT Token
```powershell
.\pasteguard.exe --text 'token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"'
```

**Expected**:
- `overall_risk: "high"`
- Finding with `type: "jwt_token"`
- `severity: "high"`
- Redacted JWT (should show `eyJh...sw5c`)

## Step 5: Test Password Assignment Detection

### Test Password Assignment
```powershell
.\pasteguard.exe --text 'password = "mySecretPassword123"'
```

**Expected**:
- `overall_risk: "high"`
- Finding with `type: "password_assignment"`
- `severity: "high"`
- Redacted password (should show `mySe...d123`)

### Test API Key
```powershell
.\pasteguard.exe --text 'api_key = "sk-1234567890abcdefghijklmnop"'
```

**Expected**: Similar detection

### Test Secret
```powershell
.\pasteguard.exe --text 'secret = "super_secret_value_12345"'
```

**Expected**: Similar detection

## Step 6: Test Token Heuristics Rule

### Test High-Entropy Token Near Keyword
```powershell
.\pasteguard.exe --text 'access_token = "AbCdEfGhIjKlMnOpQrStUvWxYz1234567890+/=AbCdEfGhIjKlMnOp"'
```

**Expected**:
- `overall_risk: "high"`
- Finding with `type: "token_heuristics"`
- `severity: "high"` or `"medium"` (depending on score)
- Redacted token (majority masked, e.g., `AbCd...MnOp`)

### Test Base64-like Token
```powershell
.\pasteguard.exe --text 'api_key = "SGVsbG9Xb3JsZFRoaXNJc0Jhc2U2NA=="'
```

**Expected**: Should detect as token_heuristics

### Test Hex-like Token
```powershell
.\pasteguard.exe --text 'secret = "deadbeef1234567890abcdef1234567890abcdef"'
```

**Expected**: Should detect as token_heuristics

## Step 7: Verify Conservative Detection (Should NOT Detect)

### Test UUID (Should be ignored)
```powershell
.\pasteguard.exe --text 'uuid = "550e8400-e29b-41d4-a716-446655440000"'
```

**Expected**: `overall_risk: "low"`, no findings

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

## Step 9: Test Multiple Findings

### Test Multiple Secrets
```powershell
.\pasteguard.exe --text "password = `"secret123`"`napi_key = `"sk-1234567890`"`ntoken = `"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U`""
```

**Expected**:
- `overall_risk: "high"`
- Multiple findings in the array
- All secrets properly redacted

## Step 10: Test with Files (Stdin)

### Create a test file
```powershell
# Create test file
@"
password = "test123"
api_key = "sk-test123456"
"@ | Out-File -FilePath test_secrets.txt -Encoding utf8

# Test with file input
Get-Content test_secrets.txt | .\pasteguard.exe
```

**Expected**: Should detect secrets from the file

## Step 11: Verify JSON Structure

### Check JSON is valid
```powershell
.\pasteguard.exe --text "test" | ConvertFrom-Json | ConvertTo-Json -Depth 10
```

**Expected**: Should output valid JSON with:
- `overall_risk` field
- `risk_rationale` field
- `findings` array
- Each finding has: `type`, `severity`, `confidence`, `reason`, `line_number`

## Step 12: Test Edge Cases

### Test Empty Input
```powershell
# Use --text= for empty string (PowerShell strips quotes from --text "")
.\pasteguard.exe --text=
```

**Expected**: `overall_risk: "low"`, empty findings

**Note**: In PowerShell, `--text ""` doesn't work because PowerShell strips the quotes. Use `--text=` instead for empty strings.

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

## Quick Verification Checklist

- [ ] Binary builds successfully
- [ ] Regular text returns low risk
- [ ] PEM keys are detected and redacted
- [ ] JWTs are detected and redacted
- [ ] Passwords are detected and redacted
- [ ] Token heuristics detects high-entropy tokens
- [ ] UUIDs are NOT detected (conservative)
- [ ] Hashes are NOT detected (conservative)
- [ ] Full secrets never appear in output
- [ ] Redacted versions appear with `...`
- [ ] Multiple findings work correctly
- [ ] JSON structure is valid
- [ ] Exit code is always 0

## Running Unit Tests

```powershell
# Run all tests
go test -v ./...

# Run specific test package
go test -v ./detector

# Run with coverage
go test ./... -cover
```

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

