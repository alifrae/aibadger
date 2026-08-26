package model

import "time"

const (
	FileKindSource = "source"
	FileKindAsset  = "asset"
	FileKindBinary = "binary"
)

type ProjectTopology struct {
	Name            string            `json:"name"`
	Languages       []string          `json:"languages"`
	PrimaryLanguage string            `json:"primary_language"`
	Stack           []string          `json:"stack"`
	Structure       string            `json:"structure"`
	Modules         []Module          `json:"modules"`
	ExternalContext []ExternalContext `json:"external_context,omitempty"`
	ScanTime        time.Duration     `json:"scan_time"`
	ProjectRoot     string            `json:"project_root"`
}

// ExternalContext represents an explicitly configured read-only context root.
// Label/Revision/Include are optional so legacy .badger-context remains valid.
type ExternalContext struct {
	Path        string                `json:"path"`
	AbsPath     string                `json:"abs_path"`
	Label       string                `json:"label,omitempty"`
	GitRevision string                `json:"git_revision,omitempty"`
	Include     []string              `json:"include,omitempty"`
	Top         []ExternalContextItem `json:"top,omitempty"`
}

type ExternalContextItem struct {
	Name  string `json:"name"`
	IsDir bool   `json:"is_dir,omitempty"`
}

type Module struct {
	Name        string        `json:"name"`
	Path        string        `json:"path"`
	FileCount   int           `json:"file_count"`
	TotalBytes  int64         `json:"total_bytes"`
	Heaviest    HeaviestFile  `json:"heaviest"`
	TopFiles    []FileSummary `json:"top_files"`
	AuxFiles    []FileSummary `json:"aux_files,omitempty"`
	SourceRoots []SourceRoot  `json:"source_roots"`
	Language    string        `json:"language"`
}

type SourceRoot struct {
	Path      string    `json:"path"`
	Role      string    `json:"role"`
	FileCount int       `json:"file_count"`
	Packages  []Package `json:"packages"`
}

type Package struct {
	Name      string        `json:"name"`
	Path      string        `json:"path"`
	FileCount int           `json:"file_count"`
	Heaviest  HeaviestFile  `json:"heaviest"`
	TopFiles  []FileSummary `json:"top_files"`
	AuxFiles  []FileSummary `json:"aux_files,omitempty"`
}

type HeaviestFile struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Size int64  `json:"size"`
	Kind string `json:"kind,omitempty"`
}

type FileSummary struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Size int64  `json:"size"`
	Kind string `json:"kind,omitempty"`
}
