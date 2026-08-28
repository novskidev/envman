package main

import (
	"fmt"
	"io"
	"regexp"
	"strings"
)

// Secret heuristics: value patterns that look like real secrets.
// Conservative — flags only high-confidence shapes.
var secretPatterns = []struct {
	name string
	re   *regexp.Regexp
}{
	{"AWS access key", regexp.MustCompile(`AKIA[0-9A-Z]{16}`)},
	{"GitHub PAT", regexp.MustCompile(`gh[pousr]_[A-Za-z0-9]{36,}`)},
	{"Slack token", regexp.MustCompile(`xox[baprs]-[A-Za-z0-9-]{10,}`)},
	{"Stripe secret", regexp.MustCompile(`sk_live_[0-9a-zA-Z]{16,}`)},
	{"JWT (HS256)", regexp.MustCompile(`eyJhbGciOi[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}`)},
	{"Generic long base64 token", regexp.MustCompile(`[A-Za-z0-9+/]{40,}={0,2}`)},
}

// keyLooksSecret: name-based heuristic (KEY/SECRET/TOKEN/PASSWORD +
// not EXAMPLE/PLACEHOLDER/DUMMY/YOUR_/XXX).
func keyLooksSecret(key string) bool {
	up := strings.ToUpper(key)
	marker := strings.Contains(up, "KEY") || strings.Contains(up, "SECRET") ||
		strings.Contains(up, "TOKEN") || strings.Contains(up, "PASSWORD") ||
		strings.Contains(up, "PASSWD") || strings.Contains(up, "PRIVATE_KEY") ||
		strings.Contains(up, "ACCESS_KEY")
	if !marker {
		return false
	}
	placeholder := strings.Contains(up, "EXAMPLE") || strings.Contains(up, "PLACEHOLDER") ||
		strings.Contains(up, "DUMMY") || strings.Contains(up, "YOUR_") ||
		strings.Contains(up, "_XXX") || strings.Contains(up, "SAMPLE") ||
		strings.Contains(up, "CHANGE_ME") || strings.Contains(up, "<") &&
			strings.Contains(up, ">")
	return marker && !placeholder
}

// SecretFinding is one suspected real secret in a file.
type SecretFinding struct {
	Key   string
	Line  string // raw line (value redacted in output)
	Probe string // which heuristic matched
}

// ScanSecrets inspects an env file for values that look like real secrets
// (useful for .env.example, which should only hold placeholders).
func ScanSecrets(f *EnvFile) []SecretFinding {
	var out []SecretFinding
	for _, key := range f.Order {
		val := f.Vars[key]
		if val == "" {
			continue
		}
		redacted := val
		if len(redacted) > 12 {
			redacted = redacted[:6] + "…" + redacted[len(redacted)-4:]
		}
		if keyLooksSecret(key) {
			out = append(out, SecretFinding{key, redacted, "key-name heuristic"})
			continue
		}
		for _, p := range secretPatterns {
			if p.re.MatchString(val) {
				out = append(out, SecretFinding{key, redacted, p.name})
				break
			}
		}
	}
	return out
}

func renderScan(w io.Writer, f *EnvFile, findings []SecretFinding) {
	if len(findings) == 0 {
		fmt.Fprintf(w, "OK: no secret-looking values in %s\n", f.Name)
		return
	}
	fmt.Fprintf(w, "Found %d possible real secret(s) in %s (should be placeholders):\n", len(findings), f.Name)
	for _, fd := range findings {
		fmt.Fprintf(w, "  %s = %s   [%s]\n", fd.Key, fd.Line, fd.Probe)
	}
}