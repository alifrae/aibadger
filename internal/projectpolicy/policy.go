package projectpolicy

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const ConfigFileName = ".badger.toml"

type SecurityPolicy struct {
	Deny         []string
	Warn         []string
	BlockSecrets bool
}

type ContextPolicy struct {
	AlwaysInclude []string
}

type DocsPolicy struct {
	CanonicalRoots []string
}

type SessionPolicy struct {
	RequireSnapshot bool
}

type WritePolicy struct {
	PatchOnly       bool
	PostApplyReview bool
}

type VerifyPolicy struct {
	// Command is an argv vector executed directly, without a shell, only after
	// the user explicitly requests verification from the post-apply screen.
	Command []string
}

type Policy struct {
	Security SecurityPolicy
	Context  ContextPolicy
	Docs     DocsPolicy
	Session  SessionPolicy
	Write    WritePolicy
	Verify   VerifyPolicy
}

func Default() Policy { return Policy{} }

func Load(root string) (Policy, error) {
	policy := Default()
	path := filepath.Join(root, ConfigFileName)
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return policy, nil
		}
		return Policy{}, fmt.Errorf("reading %s: %w", ConfigFileName, err)
	}
	defer file.Close()

	section := ""
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(stripComment(scanner.Text()))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return Policy{}, fmt.Errorf("%s:%d: expected key = value", ConfigFileName, lineNumber)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if err := assign(&policy, section, key, value); err != nil {
			return Policy{}, fmt.Errorf("%s:%d: %w", ConfigFileName, lineNumber, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return Policy{}, fmt.Errorf("reading %s: %w", ConfigFileName, err)
	}
	if err := validatePatterns(policy.Security.Deny); err != nil {
		return Policy{}, fmt.Errorf("security.deny: %w", err)
	}
	if err := validatePatterns(policy.Security.Warn); err != nil {
		return Policy{}, fmt.Errorf("security.warn: %w", err)
	}
	if len(policy.Verify.Command) > 0 && strings.TrimSpace(policy.Verify.Command[0]) == "" {
		return Policy{}, fmt.Errorf("verify.command: executable cannot be empty")
	}
	return policy, nil
}

func assign(policy *Policy, section, key, raw string) error {
	switch section + "." + key {
	case "security.deny":
		return decodePathArray(raw, &policy.Security.Deny)
	case "security.warn":
		return decodePathArray(raw, &policy.Security.Warn)
	case "security.block_secrets":
		return decodeBool(raw, &policy.Security.BlockSecrets)
	case "context.always_include":
		return decodePathArray(raw, &policy.Context.AlwaysInclude)
	case "docs.canonical_roots":
		return decodePathArray(raw, &policy.Docs.CanonicalRoots)
	case "session.require_snapshot":
		return decodeBool(raw, &policy.Session.RequireSnapshot)
	case "write.patch_only":
		return decodeBool(raw, &policy.Write.PatchOnly)
	case "write.post_apply_review":
		return decodeBool(raw, &policy.Write.PostApplyReview)
	case "verify.command":
		return decodeStringArray(raw, &policy.Verify.Command)
	default:
		return fmt.Errorf("unsupported setting %s.%s", section, key)
	}
}

func decodePathArray(raw string, dst *[]string) error {
	var values []string
	if err := decodeStringArray(raw, &values); err != nil {
		return err
	}
	for i := range values {
		values[i] = normalize(values[i])
		if values[i] == "" {
			return fmt.Errorf("paths and patterns cannot be empty")
		}
	}
	*dst = values
	return nil
}

func decodeStringArray(raw string, dst *[]string) error {
	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return fmt.Errorf("expected an array of double-quoted strings")
	}
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("array values cannot be empty")
		}
	}
	*dst = values
	return nil
}

func decodeBool(raw string, dst *bool) error {
	value, err := strconv.ParseBool(strings.TrimSpace(raw))
	if err != nil {
		return fmt.Errorf("expected true or false")
	}
	*dst = value
	return nil
}

func stripComment(line string) string {
	inString := false
	escaped := false
	for i, r := range line {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' && inString {
			escaped = true
			continue
		}
		if r == '"' {
			inString = !inString
			continue
		}
		if r == '#' && !inString {
			return line[:i]
		}
	}
	return line
}

func validatePatterns(patterns []string) error {
	for _, pattern := range patterns {
		if filepath.IsAbs(pattern) || hasParentTraversal(pattern) {
			return fmt.Errorf("unsafe pattern %q", pattern)
		}
		if _, err := compileGlob(pattern); err != nil {
			return fmt.Errorf("invalid pattern %q: %w", pattern, err)
		}
	}
	return nil
}

func (p Policy) Denies(path string) bool { return matchesAny(p.Security.Deny, path) }
func (p Policy) Warns(path string) bool  { return matchesAny(p.Security.Warn, path) }

func matchesAny(patterns []string, path string) bool {
	path = normalize(path)
	for _, pattern := range patterns {
		rx, err := compileGlob(pattern)
		if err == nil && rx.MatchString(path) {
			return true
		}
	}
	return false
}

func compileGlob(pattern string) (*regexp.Regexp, error) {
	pattern = normalize(pattern)
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				if i+2 < len(pattern) && pattern[i+2] == '/' {
					i += 2
					b.WriteString("(?:.*/)?")
				} else {
					i++
					b.WriteString(".*")
				}
			} else {
				b.WriteString("[^/]*")
			}
		case '?':
			b.WriteString("[^/]")
		default:
			b.WriteString(regexp.QuoteMeta(string(pattern[i])))
		}
	}
	b.WriteString("$")
	return regexp.Compile(b.String())
}

func normalize(path string) string {
	return strings.TrimPrefix(filepath.ToSlash(filepath.Clean(strings.TrimSpace(path))), "./")
}

func hasParentTraversal(path string) bool {
	for _, part := range strings.Split(filepath.ToSlash(path), "/") {
		if part == ".." {
			return true
		}
	}
	return false
}
