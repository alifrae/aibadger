package externalcontext

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/PVRLabs/aibadger/internal/model"
	"github.com/PVRLabs/aibadger/internal/projectpolicy"
)

// LoadNamed reads [external.<label>] sections from .badger.toml. Legacy
// .badger-context loading remains unchanged and may be used alongside these
// named sources.
func LoadNamed(projectRoot string) ([]model.ExternalContext, error) {
	policy, err := projectpolicy.Load(projectRoot)
	if err != nil {
		return nil, err
	}
	contexts := make([]model.ExternalContext, 0, len(policy.External))
	seen := map[string]bool{}
	for _, source := range policy.External {
		displayPath := filepath.ToSlash(filepath.Clean(source.Root))
		absPath, realPath, err := validateDirectory(projectRoot, displayPath)
		if err != nil {
			return nil, fmt.Errorf("external.%s: %w", source.Label, err)
		}
		key := source.Label + "\x00" + realPath
		if seen[key] {
			continue
		}
		seen[key] = true
		ctx := model.ExternalContext{
			Path:        "@" + source.Label,
			AbsPath:     realPath,
			Label:       source.Label,
			GitRevision: gitRevision(absPath),
			Include:     append([]string(nil), source.Include...),
		}
		ctx.Top = filterNamedTop(ctx, summarizeTop(absPath, ctx.Path))
		contexts = append(contexts, ctx)
	}
	sort.Slice(contexts, func(i, j int) bool { return contexts[i].Label < contexts[j].Label })
	return contexts, nil
}

// ResolveFileFiltered applies include filters after the normal safe resolver.
// Callers that support named policy sources should use this function.
func ResolveFileFiltered(projectRoot string, contexts []model.ExternalContext, requestPath string) FileResolution {
	resolution := ResolveFile(projectRoot, contexts, requestPath)
	if len(resolution.Matches) == 0 {
		return resolution
	}
	filtered := make([]FileMatch, 0, len(resolution.Matches))
	for _, match := range resolution.Matches {
		if IsAllowedPath(match.Context, match.RelPath, false) {
			filtered = append(filtered, match)
		}
	}
	return FileResolution{Matches: filtered}
}

// IsDisplayPath reports whether a path addresses a named read-only external
// source. Both @label/path and label/path are recognized because tagged-file
// parsing consumes the leading @ before resolution.
func IsDisplayPath(contexts []model.ExternalContext, path string) bool {
	path = filepath.ToSlash(filepath.Clean(strings.TrimSpace(path)))
	for _, ctx := range contexts {
		if ctx.Label == "" {
			continue
		}
		for _, prefix := range []string{"@" + ctx.Label, ctx.Label} {
			if path == prefix || strings.HasPrefix(path, prefix+"/") {
				return true
			}
		}
	}
	return false
}

// IsAllowedPath applies a named source's include filter. Legacy external
// contexts have no Include list and therefore preserve their existing behavior.
func IsAllowedPath(ctx model.ExternalContext, rel string, isDir bool) bool {
	if len(ctx.Include) == 0 {
		return true
	}
	rel = filepath.ToSlash(filepath.Clean(rel))
	for _, pattern := range ctx.Include {
		if projectpolicy.MatchGlob(pattern, rel) {
			return true
		}
		if isDir {
			prefix := strings.TrimSuffix(rel, "/") + "/"
			plain := strings.TrimPrefix(filepath.ToSlash(pattern), "./")
			if strings.HasPrefix(plain, prefix) {
				return true
			}
		}
	}
	return false
}

func filterNamedTop(ctx model.ExternalContext, items []model.ExternalContextItem) []model.ExternalContextItem {
	if len(ctx.Include) == 0 {
		return items
	}
	filtered := make([]model.ExternalContextItem, 0, len(items))
	for _, item := range items {
		if IsAllowedPath(ctx, item.Name, item.IsDir) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func gitRevision(root string) string {
	cmd := exec.Command("git", "-C", root, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
