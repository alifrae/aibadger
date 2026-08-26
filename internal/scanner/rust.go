package scanner

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/PVRLabs/aibadger/internal/model"
	"github.com/PVRLabs/aibadger/internal/promptpolicy"
)

// RustDetector discovers Cargo packages and maps common Rust source layouts
// into Badger modules. Virtual workspace manifests are intentionally not
// emitted as crates unless they also contain a [package] section.
type RustDetector struct {
	Exclusions map[string]bool
}

func NewRustDetector() *RustDetector {
	return &RustDetector{Exclusions: cloneExclusions(commonIgnoredDirs, ".cargo", "vendor")}
}

func (r *RustDetector) Detect(root string) ([]model.Module, error) {
	manifests, err := discoverProjectMarkers(root, "Cargo.toml")
	if err != nil {
		return nil, err
	}
	var modules []model.Module
	for _, manifest := range manifests {
		name, hasPackage := cargoPackageName(manifest)
		if !hasPackage {
			continue
		}
		crateRoot := filepath.Dir(manifest)
		module := r.analyzeCrate(root, crateRoot, name)
		if module.FileCount > 0 || rustFileExists(filepath.Join(crateRoot, "build.rs")) {
			modules = append(modules, module)
		}
	}
	sort.Slice(modules, func(i, j int) bool { return modules[i].Path < modules[j].Path })
	return modules, nil
}

func cargoPackageName(manifest string) (string, bool) {
	f, err := os.Open(manifest)
	if err != nil {
		return "", false
	}
	defer f.Close()
	section := ""
	hasPackage := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
			if section == "package" {
				hasPackage = true
			}
			continue
		}
		if section != "package" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(key) != "name" {
			continue
		}
		name := strings.TrimSpace(value)
		if hash := strings.Index(name, "#"); hash >= 0 {
			name = strings.TrimSpace(name[:hash])
		}
		name = strings.Trim(name, "\"")
		if name != "" {
			return name, true
		}
	}
	return "", hasPackage
}

func (r *RustDetector) analyzeCrate(projectRoot, crateRoot, name string) model.Module {
	if name == "" {
		name = filepath.Base(crateRoot)
	}
	module := model.Module{Name: name, Path: relativePath(projectRoot, crateRoot), Language: "Rust"}

	roots := []struct {
		name string
		role string
	}{
		{"src", "Main Source"},
		{"tests", "Test Source"},
		{"benches", "Benchmark"},
		{"examples", "Examples"},
	}
	for _, spec := range roots {
		full := filepath.Join(crateRoot, spec.name)
		info, err := os.Stat(full)
		if err != nil || !info.IsDir() {
			continue
		}
		sr, bytes := r.scanRustRoot(projectRoot, full, spec.role)
		if sr.FileCount == 0 {
			continue
		}
		module.SourceRoots = append(module.SourceRoots, sr)
		module.FileCount += sr.FileCount
		module.TotalBytes += bytes
		for _, pkg := range sr.Packages {
			for _, file := range pkg.TopFiles {
				module.TopFiles = addRustTopFile(module.TopFiles, file, moduleTopFileLimit(module.Path, maxPackageTopFiles))
			}
		}
	}

	buildPath := filepath.Join(crateRoot, "build.rs")
	if info, err := os.Stat(buildPath); err == nil && info.Mode().IsRegular() && !promptpolicy.IsSensitivePath(relativePath(projectRoot, buildPath)) {
		file := model.FileSummary{Name: "build.rs", Path: relativePath(projectRoot, buildPath), Size: info.Size(), Kind: model.FileKindSource}
		module.FileCount++
		module.TotalBytes += info.Size()
		module.TopFiles = addRustTopFile(module.TopFiles, file, moduleTopFileLimit(module.Path, maxPackageTopFiles))
	}
	if len(module.TopFiles) > 0 {
		module.Heaviest = heaviestFromSummary(module.TopFiles[0])
	}
	return module
}

func (r *RustDetector) scanRustRoot(projectRoot, fullRoot, role string) (model.SourceRoot, int64) {
	sr := model.SourceRoot{Path: relativePath(projectRoot, fullRoot), Role: role}
	packages := map[string]*model.Package{}
	var totalBytes int64
	_ = filepath.WalkDir(fullRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != fullRoot && shouldSkipDir(d.Name(), r.Exclusions) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.ToLower(filepath.Ext(d.Name())) != ".rs" || promptpolicy.IsSensitivePath(relativePath(projectRoot, path)) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		sr.FileCount++
		totalBytes += info.Size()
		pkgDir := filepath.Dir(path)
		pkgPath := relativePath(projectRoot, pkgDir)
		pkg := packages[pkgPath]
		if pkg == nil {
			pkg = &model.Package{Name: rustPackageName(relativePath(fullRoot, pkgDir)), Path: pkgPath}
			packages[pkgPath] = pkg
		}
		pkg.FileCount++
		file := model.FileSummary{Name: d.Name(), Path: relativePath(projectRoot, path), Size: info.Size(), Kind: model.FileKindSource}
		pkg.TopFiles = addRustTopFile(pkg.TopFiles, file, packageTopFileLimit(pkg.Path, maxPackageTopFiles))
		if len(pkg.TopFiles) > 0 {
			pkg.Heaviest = heaviestFromSummary(pkg.TopFiles[0])
		}
		return nil
	})
	for _, pkg := range packages {
		sr.Packages = append(sr.Packages, *pkg)
	}
	sort.Slice(sr.Packages, func(i, j int) bool { return sr.Packages[i].Path < sr.Packages[j].Path })
	return sr, totalBytes
}

func rustPackageName(rel string) string {
	rel = filepath.ToSlash(filepath.Clean(rel))
	if rel == "" || rel == "." {
		return "root"
	}
	return rel
}

func addRustTopFile(files []model.FileSummary, file model.FileSummary, limit int) []model.FileSummary {
	files = append(files, file)
	sort.Slice(files, func(i, j int) bool {
		ri, rj := rustFileRank(files[i].Name), rustFileRank(files[j].Name)
		if ri != rj {
			return ri < rj
		}
		if files[i].Size != files[j].Size {
			return files[i].Size > files[j].Size
		}
		return files[i].Path < files[j].Path
	})
	if limit > 0 && len(files) > limit {
		return files[:limit]
	}
	return files
}

func rustFileRank(name string) int {
	switch strings.ToLower(name) {
	case "lib.rs", "main.rs":
		return 0
	case "mod.rs", "build.rs":
		return 1
	default:
		return 2
	}
}

func rustFileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}
