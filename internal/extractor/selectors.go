package extractor

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/PVRLabs/aibadger/internal/filekind"
	"github.com/PVRLabs/aibadger/internal/model"
	"github.com/PVRLabs/aibadger/internal/promptpolicy"
)

const (
	maxSelectorMatches   = 12
	maxTestSelectorMatch = 8
	maxSelectorFileBytes = 1024 * 1024
	maxSelectorScanFiles = 5000
)

func (e *Extractor) expandCommands(commands []Command) ([]Command, error) {
	var expanded []Command
	seen := map[string]bool{}
	for _, cmd := range commands {
		var resolved []Command
		switch cmd.Type {
		case "SYMBOL":
			if cmd.Path == "" || cmd.Pattern == "" {
				return nil, fmt.Errorf("SYMBOL requires path#symbol")
			}
			resolved = []Command{{Type: "NEAR", Path: cmd.Path, Pattern: cmd.Pattern}}
		case "REFERENCES":
			matches, err := e.searchProject(cmd.Path, false, maxSelectorMatches)
			if err != nil {
				return nil, err
			}
			resolved = matches
		case "TESTS":
			matches, err := e.searchProject(cmd.Path, true, maxTestSelectorMatch)
			if err != nil {
				return nil, err
			}
			resolved = matches
		case "SEARCH":
			matches, err := e.searchProject(cmd.Path, false, maxSelectorMatches)
			if err != nil {
				return nil, err
			}
			resolved = matches
		default:
			resolved = []Command{cmd}
		}
		if len(resolved) == 0 && isDiscoveryCommand(cmd.Type) {
			return nil, fmt.Errorf("%s found no bounded matches for %q", cmd.Type, cmd.Path)
		}
		for _, item := range resolved {
			key := item.Type + "\x00" + item.Path + "\x00" + item.Pattern
			if seen[key] {
				continue
			}
			seen[key] = true
			expanded = append(expanded, item)
		}
	}
	return expanded, nil
}

func isDiscoveryCommand(kind string) bool {
	switch kind {
	case "REFERENCES", "TESTS", "SEARCH":
		return true
	default:
		return false
	}
}

func (e *Extractor) searchProject(literal string, testsOnly bool, limit int) ([]Command, error) {
	literal = strings.TrimSpace(strings.Trim(literal, "\""))
	if literal == "" {
		return nil, fmt.Errorf("search selector requires a non-empty literal")
	}
	if limit <= 0 {
		limit = maxSelectorMatches
	}
	type hit struct {
		path string
	}
	var hits []hit
	scanned := 0
	err := filepath.WalkDir(e.ProjectRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != e.ProjectRoot && shouldSkipSelectorDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if len(hits) >= limit || scanned >= maxSelectorScanFiles {
			return nil
		}
		rel, relErr := filepath.Rel(e.ProjectRoot, path)
		if relErr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if promptpolicy.IsSensitivePath(rel) || (testsOnly && !isLikelyTestPath(rel)) {
			return nil
		}
		kind := filekind.Classify(path)
		if kind == model.FileKindBinary || kind == model.FileKindAsset {
			return nil
		}
		info, infoErr := d.Info()
		if infoErr != nil || info.Size() > maxSelectorFileBytes {
			return nil
		}
		scanned++
		data, readErr := os.ReadFile(path)
		if readErr != nil || !strings.Contains(string(data), literal) {
			return nil
		}
		hits = append(hits, hit{path: rel})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].path < hits[j].path })
	commands := make([]Command, 0, len(hits))
	for _, hit := range hits {
		commands = append(commands, Command{Type: "NEAR", Path: hit.path, Pattern: literal})
	}
	return commands, nil
}

func shouldSkipSelectorDir(name string) bool {
	switch strings.ToLower(name) {
	case ".git", ".hg", ".svn", "node_modules", "target", "build", "dist", ".venv", "venv", "__pycache__", ".mypy_cache", ".pytest_cache":
		return true
	default:
		return false
	}
}

func isLikelyTestPath(path string) bool {
	lower := strings.ToLower(filepath.ToSlash(path))
	base := strings.ToLower(filepath.Base(lower))
	if strings.Contains(lower, "/tests/") || strings.Contains(lower, "/test/") || strings.Contains(lower, "/__tests__/") {
		return true
	}
	if strings.HasPrefix(base, "test_") || strings.Contains(base, "_test.") || strings.Contains(base, ".test.") || strings.Contains(base, ".spec.") {
		return true
	}
	return strings.HasSuffix(base, "test.rs") || strings.HasSuffix(base, "tests.rs")
}
