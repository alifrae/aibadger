package egress

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/PVRLabs/aibadger/internal/projectpolicy"
)

type Decision struct {
	Blocked bool
	Warning string
}

var highConfidenceSecretPatterns = []struct {
	name string
	rx   *regexp.Regexp
}{
	{"private key", regexp.MustCompile(`(?i)-----BEGIN(?: [A-Z0-9]+)? PRIVATE KEY-----`)},
	{"AWS access key", regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`)},
	{"GitHub token", regexp.MustCompile(`\b(?:gh[pousr]_[A-Za-z0-9]{20,}|github_pat_[A-Za-z0-9_]{20,})\b`)},
	{"OpenAI-style API key", regexp.MustCompile(`\bsk-[A-Za-z0-9_-]{20,}\b`)},
	{"Google API key", regexp.MustCompile(`\bAIza[0-9A-Za-z_-]{35}\b`)},
}

var assignmentPattern = regexp.MustCompile(`(?i)(?:api[_-]?key|access[_-]?token|auth[_-]?token|client[_-]?secret|password|passwd)\s*[:=]\s*["']([^"']{20,})["']`)

func Inspect(path, content string, policy projectpolicy.Policy) Decision {
	if policy.Denies(path) {
		return Decision{Blocked: true, Warning: fmt.Sprintf("%s: blocked by .badger.toml security.deny", path)}
	}
	if policy.Security.BlockSecrets {
		if kind := detectSecret(content); kind != "" {
			return Decision{Blocked: true, Warning: fmt.Sprintf("%s: blocked by egress DLP (%s detected)", path, kind)}
		}
	}
	if policy.Warns(path) {
		return Decision{Warning: fmt.Sprintf("%s: matches .badger.toml security.warn; review before copying", path)}
	}
	return Decision{}
}

func detectSecret(content string) string {
	for _, candidate := range highConfidenceSecretPatterns {
		if candidate.rx.FindStringIndex(content) != nil {
			return candidate.name
		}
	}
	for _, match := range assignmentPattern.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 && looksLikeSecret(match[1]) {
			return "credential assignment"
		}
	}
	return ""
}

func looksLikeSecret(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	for _, placeholder := range []string{"example", "placeholder", "changeme", "your_", "your-", "dummy", "test", "<", "${"} {
		if strings.Contains(lower, placeholder) {
			return false
		}
	}
	var letters, digits, other int
	for _, r := range value {
		switch {
		case unicode.IsLetter(r):
			letters++
		case unicode.IsDigit(r):
			digits++
		default:
			other++
		}
	}
	classes := 0
	if letters > 0 {
		classes++
	}
	if digits > 0 {
		classes++
	}
	if other > 0 {
		classes++
	}
	return len(value) >= 24 && classes >= 2
}
