package extractor

import (
	"errors"
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

var errSelectorSearchComplete = errors.New("selector search complete")

func (e *Extractor) expandCommands(commands []Command) ([]Command, []string) {
	var expanded []Command
	var failures []string
	seen := map[string]bool{}
	for _, cmd := range commands {
		var resolved []Command
		switch cmd.Type {
		case "SYMBOL":
			if cmd.Path == "" || cmd.Pattern == "" {
				failures = append(failures, fmt.Sprintf("%s: SYMBOL requires path#symbol", cmd.Path))
				continue
			}
			resolved = []Command{{Type: "NEAR", Path: cmd.Path, Pattern: cmd.Pattern}}
		case "REFERENCES":
			matches, err := e.searchProject(cmd.Path, false, maxSelectorMatches)
			if err != nil {
				failures = append(failures, fmt.Sprintf("%s: %v", cmd.Path, err))
				continue
			}
			resolved = matches
		case "TESTS":
			matches, err := e.searchProject(cmd.Path, true, maxTestSelectorMatch)
			if err != nil {
				failures = append(failures, fmt.Sprintf("%s: %v", cmd.Path, err))
				continue
			}
			resolved = matches
		case "SEARCH":
			matches, err := e.searchProject(cmd.Path, false, maxSelectorMatches)
			if err != nil {
				failures = append(failures, fmt.Sprintf("%s: %v", cmd.Path, err))
				continue
			}
			resolved = matches
		default:
			resolved = []Command{cmd}
		}
		if len(resolved) == 0 && isDiscoveryCommand(cmd.Type) {
			failures = append(failures, fmt.Sprintf("%s: %s found no bounded matches", cmd.Path, cmd.Type))
			continue
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
	return expanded, failures
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
		if d.Type()&os.ModeSymlink != 0 {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if path != e.ProjectRoot && shouldSkipSelectorDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if len(hits) >= limit || scanned >= maxSelectorScanFiles {
			return errSelectorSearchComplete
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
	if err != nil && !errors.Is(err, errSelectorSearchComplete) {
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
	for _, prefix := range []string{"tests/", "test/", "__tests__/"} {
		if strings.HasPrefix(lower, prefix) || strings.Contains(lower, "/"+prefix) {
			return true
		}
	}
	if strings.HasPrefix(base, "test_") || strings.Contains(base, "_test.") || strings.Contains(base, ".test.") || strings.Contains(base, ".spec.") {
		return true
	}
	return strings.HasSuffix(base, "test.rs") || strings.HasSuffix(base, "tests.rs")
}
