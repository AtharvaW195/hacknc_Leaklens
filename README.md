# Pasteguard 🔒

[![Go Version](https://img.shields.io/badge/go-1.21+-00ADD8?style=flat-square&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)](LICENSE)
[![Tests](https://img.shields.io/badge/tests-95%2B-passing-brightgreen?style=flat-square)](https://github.com/yourusername/pasteguard)
[![Coverage](https://img.shields.io/badge/coverage-95%25-brightgreen?style=flat-square)](https://github.com/yourusername/pasteguard)
[![Standard Library](https://img.shields.io/badge/stdlib-only-success?style=flat-square)](https://golang.org/pkg/)

> A lightweight, fast secret detection tool that scans text for sensitive information like passwords, API keys, JWT tokens, and private keys. Available as both a CLI tool and HTTP API server.

## ✨ Features

- 🔍 **Multiple Detection Rules**
  - PEM private keys (RSA, EC, DSA)
  - JWT tokens
  - Password assignments
  - High-entropy token detection (conservative)

- 🛡️ **Security First**
  - Automatic secret redaction (never leaks full secrets)
  - No input logging
  - Rate limiting (HTTP mode)
  - Request size limits

- 🚀 **Dual Mode Operation**
  - **CLI Mode**: Analyze text from command line or stdin
  - **HTTP Server Mode**: REST API for programmatic access

- ⚡ **Fast & Lightweight**
  - Standard library only (no external dependencies)
  - Single binary deployment
  - Deterministic output

- 🧪 **Well Tested**
  - 95+ unit tests
  - 95%+ code coverage
  - Comprehensive test suite

## 📦 Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/yourusername/pasteguard.git
cd pasteguard

# Build
go build -o pasteguard .

# Or install globally
go install .
```

### Download Binary

Download pre-built binaries from the [Releases](https://github.com/yourusername/pasteguard/releases) page.

## 🚀 Quick Start

### CLI Mode

```bash
# Analyze text from command line
pasteguard --text "password = secret123"

# Analyze from stdin
echo "api_key = sk-1234567890" | pasteguard

# Analyze a file
cat config.txt | pasteguard
```

**Example Output:**
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

### HTTP Server Mode

```bash
# Start the server (default port :8787)
pasteguard serve

# Start on custom port
pasteguard serve --addr :8080
```

**Test the API:**
```bash
# Health check
curl http://localhost:8787/health

# Analyze text
curl -X POST http://localhost:8787/analyze \
  -H "Content-Type: application/json" \
  -d '{"text": "password = secret123"}'
```

## 📖 Usage

### CLI Mode

```bash
# Basic usage
pasteguard --text "your text here"

# Empty string (handled correctly)
pasteguard --text ""

# Pipe from stdin
echo "your text" | pasteguard

# From file
cat file.txt | pasteguard
```

### HTTP Server Mode

#### Start Server
```bash
pasteguard serve --addr :8787
```

#### Endpoints

**GET /health**
```bash
curl http://localhost:8787/health
```
Response:
```json
{"status": "ok"}
```

**POST /analyze**
```bash
curl -X POST http://localhost:8787/analyze \
  -H "Content-Type: application/json" \
  -d '{"text": "password = secret123"}'
```

Response:
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

#### PowerShell Example
```powershell
$body = @{
    text = "password = `"secret123`""
} | ConvertTo-Json

Invoke-RestMethod -Uri http://localhost:8787/analyze `
  -Method POST `
  -ContentType "application/json" `
  -Body $body
```

## 🔍 Detection Rules

### PEM Private Keys
Detects RSA, EC, DSA, and generic private keys in PEM format.

```bash
pasteguard --text "-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA...
-----END RSA PRIVATE KEY-----"
```

### JWT Tokens
Detects JSON Web Tokens (3-part base64 format).

```bash
pasteguard --text 'token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."'
```

### Password Assignments
Detects common password/secret assignment patterns.

```bash
pasteguard --text 'password = "secret123"'
pasteguard --text 'api_key = "sk-1234567890"'
pasteguard --text 'secret: "my_secret_value"'
```

### Token Heuristics
Detects high-entropy token-like strings with conservative filtering (ignores UUIDs, hashes, commit hashes).

```bash
pasteguard --text 'api_key = "AbCdEfGhIjKlMnOpQrStUvWxYz1234567890"'
```

## 🛡️ Security Features

### Secret Redaction
All secrets are automatically redacted in the output. Full secrets never appear in JSON responses.

- **Token heuristics**: >50% of token masked
- **Other rules**: First 4 and last 4 characters shown

### HTTP Server Security
- **Rate Limiting**: 100 requests per minute per IP
- **Size Limits**: 1MB maximum request body
- **No Input Logging**: User input never logged to console

## 📊 Response Format

All responses follow this JSON structure:

```json
{
  "overall_risk": "high" | "medium" | "low",
  "risk_rationale": "Description of risk level",
  "findings": [
    {
      "type": "pem_private_key" | "jwt_token" | "password_assignment" | "token_heuristics",
      "severity": "high" | "medium" | "low",
      "confidence": "high" | "medium" | "low",
      "reason": "redacted_secret",
      "line_number": 1
    }
  ]
}
```

### Risk Levels
- **high**: Any finding with high severity detected
- **medium**: Findings detected but none are high severity
- **low**: No findings detected

## 🧪 Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test ./... -cover

# Run specific test suites
go test -v ./... -run TestCLI
go test -v ./server
go test -v ./detector
```

**Test Coverage:**
- CLI Tests: 13 tests
- HTTP Server Tests: 15 tests
- Rule Tests: 50+ tests
- Engine Tests: 10+ tests
- Redaction Tests: 8 tests
- Merge/Sort Tests: 11 tests
- **Total: 95+ tests**

## 📚 Documentation

- [Architecture Documentation](ARCHITECTURE.md) - System architecture and design
- [Testing Guide](TESTING_GUIDE.md) - Comprehensive testing instructions
- [ASCII Architecture Diagram](architecture-diagram.txt) - Text-based architecture diagram

## 🏗️ Architecture

Pasteguard uses a modular rule-based architecture:

```
Entry Points (CLI/HTTP)
    ↓
Detector Engine
    ↓
Detection Rules (PEM, JWT, Password, Token Heuristics)
    ↓
Processing Pipeline (Merge, Sort, Score, Redact)
    ↓
JSON Output
```

See [ARCHITECTURE.md](ARCHITECTURE.md) for detailed architecture documentation.

## 🔧 Development

### Prerequisites
- Go 1.21 or later

### Build
```bash
go build -o pasteguard .
```

### Run Tests
```bash
go test ./...
```

### Project Structure
```
pasteguard/
├── main.go              # CLI entry point
├── detector/            # Detection engine and rules
│   ├── engine.go       # Core engine
│   ├── rule.go         # Rule interface
│   ├── pem_rule.go      # PEM detection
│   ├── jwt_rule.go      # JWT detection
│   ├── password_rule.go # Password detection
│   └── token_heuristics_rule.go # Token detection
├── server/              # HTTP server
│   └── server.go       # HTTP handlers
└── *_test.go           # Test files
```

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Built with Go standard library only
- Inspired by secret scanning tools like GitGuardian, TruffleHog, and Gitleaks

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/yourusername/pasteguard/issues)
- **Discussions**: [GitHub Discussions](https://github.com/yourusername/pasteguard/discussions)

---

**Made with ❤️ using Go**
