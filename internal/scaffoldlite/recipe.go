package scaffoldlite

import (
	"io/fs"
	"time"
)

type Recipe struct {
	ScaffoldVersion string         `yaml:"scaffold_version" json:"scaffold_version"`
	TemplatesDir    string         `yaml:"templates_dir" json:"templates_dir"`
	Vars            map[string]any `yaml:"vars,omitempty" json:"vars,omitempty"`
	Files           []FileEntry    `yaml:"files" json:"files"`
}

type FileEntry struct {
	Path     string         `yaml:"path" json:"path"`
	Template string         `yaml:"template" json:"template"`
	With     map[string]any `yaml:"with,omitempty" json:"with,omitempty"`
}

type Options struct {
	TemplatesFS  fs.FS
	ScaffoldPath string
	Dest         string
	Dry          bool
	Overwrite    bool
	Vars         map[string]any
}

type ParseOptions struct {
	StrictMode        bool
	MaxFileSize       int64
	MaxParseTime      time.Duration
	ValidateSchema    bool
	ValidateSemantics bool
	AllowExtensions   bool
}

var DefaultParseOptions = ParseOptions{
	StrictMode:        true,
	MaxFileSize:       1024 * 1024,
	MaxParseTime:      time.Second,
	ValidateSchema:    true,
	ValidateSemantics: true,
	AllowExtensions:   false,
}

type RenderContext struct {
	Vars   map[string]any
	File   FileEntry
	Recipe *Recipe
}

type ScaffoldStats struct {
	FilesGenerated  int
	FilesSkipped    int
	FilesFailed     int
	TotalFiles      int
	Duration        time.Duration
	TemplatesParsed int
}
