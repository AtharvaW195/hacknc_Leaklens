# PowerShell Usage Examples

## The Problem

In PowerShell, `curl` is an alias for `Invoke-WebRequest`, which has different syntax than Unix `curl`. This causes errors when using Unix curl syntax.

## Solutions

### Option 1: Use Invoke-RestMethod (Recommended)

This is the PowerShell-native way and works best:

```powershell
# Health check
Invoke-RestMethod -Uri http://localhost:8787/health

# Analyze text
$body = @{
    text = "password = secret123"
} | ConvertTo-Json

Invoke-RestMethod -Uri http://localhost:8787/analyze `
  -Method POST `
  -ContentType "application/json" `
  -Body $body
```

### Option 2: Use curl.exe explicitly

If you have curl.exe installed (Windows 10+ includes it), use `curl.exe` instead of `curl`:

```powershell
# Health check
curl.exe http://localhost:8787/health

# Analyze text (note the escaped quotes)
curl.exe -X POST http://localhost:8787/analyze `
  -H "Content-Type: application/json" `
  -d "{\"text\": \"password = secret123\"}"
```

### Option 3: Remove the curl alias temporarily

Remove the alias for the current session:

```powershell
Remove-Alias curl -ErrorAction SilentlyContinue
# Now curl will use curl.exe if available
curl -X POST http://localhost:8787/analyze -H "Content-Type: application/json" -d '{"text": "password = secret123"}'
```

## Complete Examples

### Health Check

**PowerShell (Invoke-RestMethod):**
```powershell
Invoke-RestMethod -Uri http://localhost:8787/health
```

**PowerShell (curl.exe):**
```powershell
curl.exe http://localhost:8787/health
```

### Analyze Text

**PowerShell (Invoke-RestMethod) - Recommended:**
```powershell
$body = @{
    text = "password = secret123"
} | ConvertTo-Json

$response = Invoke-RestMethod -Uri http://localhost:8787/analyze `
  -Method POST `
  -ContentType "application/json" `
  -Body $body

# Pretty print the response
$response | ConvertTo-Json -Depth 10
```

**PowerShell (curl.exe):**
```powershell
curl.exe -X POST http://localhost:8787/analyze `
  -H "Content-Type: application/json" `
  -d "{\"text\": \"password = secret123\"}"
```

### Analyze with PEM Key

**PowerShell (Invoke-RestMethod):**
```powershell
$pemKey = @"
-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAyK8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8v
-----END RSA PRIVATE KEY-----
"@

$body = @{
    text = $pemKey
} | ConvertTo-Json

Invoke-RestMethod -Uri http://localhost:8787/analyze `
  -Method POST `
  -ContentType "application/json" `
  -Body $body
```

### Analyze with JWT

**PowerShell (Invoke-RestMethod):**
```powershell
$body = @{
    text = 'token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"'
} | ConvertTo-Json

Invoke-RestMethod -Uri http://localhost:8787/analyze `
  -Method POST `
  -ContentType "application/json" `
  -Body $body
```

## Tips

1. **Always use `Invoke-RestMethod` for JSON APIs** - It automatically parses JSON responses
2. **Use `ConvertTo-Json` for request bodies** - Handles escaping automatically
3. **Use backticks (`) for line continuation** in PowerShell
4. **Use `curl.exe` if you prefer curl syntax** - But remember to escape quotes properly

