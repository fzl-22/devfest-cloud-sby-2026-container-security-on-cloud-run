# Layer 1 — Application Security Checks

Covers: source code, secrets handling, input validation, error handling, logging.
Language examples use Go, but the patterns apply to any language.

---

## A01 — Broken Access Control + SSRF

**Checks:**
- Are all routes protected by appropriate authorization middleware?
- Does the app make HTTP requests to user-controlled URLs? (SSRF)
- Are path traversal attacks possible? (e.g., `../../etc/passwd`)

```go
// BAD — user-controlled URL fetched server-side
url := r.URL.Query().Get("callback")
resp, _ := http.Get(url)  // ← SSRF

// GOOD — allowlist of permitted domains
allowedHosts := map[string]bool{"api.trusted.com": true}
```

**Fix:** Validate and allowlist all outbound URLs. Use `net/url` to parse and check hostname.

---

## A02 — Security Misconfiguration

**Checks:**
- Is `DEBUG` mode driven by an env var and correctly disabled in non-dev environments?
- Are default framework settings (CORS, rate limit, timeouts) explicitly overridden?
- Are stack trace / internal errors exposed to callers?

```go
// BAD
debug := true  // hardcoded

// GOOD
debug := os.Getenv("DEBUG") == "true"
```

---

## A03 — Software Supply Chain Failures

**Checks:**
- Are all dependencies pinned to exact versions in `go.mod` / `package.json`?
- Is there a lock file (`go.sum`, `package-lock.json`) committed?
- Are dependencies scanned for known CVEs?

```bash
# Go: scan for known vulnerabilities
govulncheck ./...

# Check for outdated dependencies
go list -u -m all
```

**Fix:** Add `govulncheck` to CI pipeline. Treat HIGH/CRITICAL CVEs as build failures.

---

## A04 — Cryptographic Failures

**Checks:**
- Are secrets read from env vars / Secret Manager, never hardcoded?
- Are JWT signing keys strong (≥256-bit) and rotated?
- Is sensitive data encrypted at rest and in transit?
- Could secrets be accidentally logged?

```go
// BAD — hardcoded
adminKey := "dev-test-admin-secret-key-3b9e4a1"

// GOOD — from environment (injected by Secret Manager at runtime)
adminKey := os.Getenv("TODO_ADMIN_KEY")
if adminKey == "" {
    log.Fatal("TODO_ADMIN_KEY is required")
}
```

```go
// BAD — secret in log
log.Printf("authenticating with key: %s", adminKey)

// GOOD
log.Printf("authenticating with key: [REDACTED]")
```

**Fix:** Use `google.golang.org/api/secretmanager` client to load secrets at startup.
Never log values that come from secret-bearing env vars.

---

## A05 — Injection

**Checks:**
- Is user input validated and sanitized before use in queries, commands, or templates?
- Are SQL queries using parameterized statements?
- Is there any `exec.Command` or `os/exec` with user-controlled input?

```go
// BAD — SQL injection
query := "SELECT * FROM todos WHERE id = " + userID

// GOOD — parameterized
row := db.QueryRow("SELECT * FROM todos WHERE id = $1", userID)
```

```go
// BAD — command injection
out, _ := exec.Command("sh", "-c", "echo "+userInput).Output()

// GOOD — no shell, direct args
out, _ := exec.Command("echo", userInput).Output()
```

---

## A06 — Insecure Design

**Checks:**
- Is there a rate-limiting middleware on public endpoints?
- Are timeouts set on all outbound HTTP clients?
- Is input length/size validated (protection against large payload DoS)?

```go
// BAD — no timeout
client := &http.Client{}

// GOOD
client := &http.Client{Timeout: 10 * time.Second}
```

```go
// Rate limiting with Fiber
app.Use(limiter.New(limiter.Config{
    Max:        20,
    Expiration: 30 * time.Second,
}))
```

---

## A07 — Authentication Failures

**Checks:**
- Are admin/sensitive routes protected with authentication middleware?
- Are JWT tokens validated (signature, expiry, issuer, audience)?
- Is there brute-force protection on auth endpoints?
- Are API keys rotatable without redeployment?

```go
// BAD — static key comparison with no expiry
if r.Header.Get("X-Admin-Key") == os.Getenv("TODO_ADMIN_KEY") { ... }

// GOOD — JWT with expiry + signing key validation
token, err := jwt.Parse(tokenStr, keyFunc)
if err != nil || !token.Valid { ... }
```

---

## A08 — Software & Data Integrity Failures

**Checks:**
- Is request body size limited?
- Are file uploads validated (type, size, content)?
- Is data from external sources verified before processing?

```go
// Limit request body size (Fiber)
app := fiber.New(fiber.Config{
    BodyLimit: 1 * 1024 * 1024, // 1MB
})
```

---

## A09 — Security Logging & Alerting Failures

**Checks:**
- Are logs structured (JSON) and sent to stdout?
- Do logs include request ID, user ID, action — but NOT secrets or PII?
- Are authentication failures logged with enough context to alert on?
- Are 5xx errors logged with full context server-side?

```go
// BAD — unstructured, leaks key
log.Printf("request from %s with key %s", ip, key)

// GOOD — structured JSON, no secrets
log.Info().
    Str("request_id", reqID).
    Str("ip", ip).
    Str("method", r.Method).
    Str("path", r.URL.Path).
    Int("status", status).
    Msg("request handled")
```

**Fix:** Use `zerolog`, `zap`, or `slog` for structured logging. Never log secrets.

---

## A10 — Mishandling of Exceptional Conditions

**Checks:**
- Do unhandled panics expose stack traces to HTTP callers?
- Is there a recover middleware that catches panics and returns a generic 500?
- Does the app handle `SIGTERM` for graceful shutdown?
- Are all errors checked (no silent `_` discards on security-sensitive operations)?

```go
// BAD — panic leaks internals to caller
func handler(c *fiber.Ctx) error {
    data := riskyOperation() // panics → exposed stack trace
    return c.JSON(data)
}

// GOOD — recover middleware
app.Use(recover.New(recover.Config{
    EnableStackTrace: false, // never send stack to client
}))
```

```go
// Graceful shutdown on SIGTERM
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
app.ShutdownWithContext(ctx)
```
