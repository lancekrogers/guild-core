package scaffoldlite

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

type Renderer interface {
	RenderTemplate(ctx context.Context, templateName string, context RenderContext) ([]byte, error)
	RenderRecipe(ctx context.Context, recipe *Recipe, options Options) (*ScaffoldStats, error)
	SetTemplateFS(fsys fs.FS)
}

type templateRenderer struct {
	templateFS    fs.FS
	templateCache *TemplateCache
	funcMap       template.FuncMap
	fileSystem    FileSystem
}

func NewRenderer(fsys fs.FS, fileSystem FileSystem) Renderer {
	return &templateRenderer{
		templateFS:    fsys,
		templateCache: NewTemplateCache(100, 10*time.Minute),
		funcMap:       getTemplateFuncMap(),
		fileSystem:    fileSystem,
	}
}

func (tr *templateRenderer) SetTemplateFS(fsys fs.FS) {
	tr.templateFS = fsys
	tr.templateCache.Clear()
}

func (tr *templateRenderer) RenderTemplate(ctx context.Context, templateName string, context RenderContext) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled before rendering")
	}
	tmpl, err := tr.getTemplate(ctx, templateName, context.Recipe.TemplatesDir)
	if err != nil {
		return nil, ErrTemplateRender(templateName, err)
	}
	data := tr.prepareTemplateData(context)
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, ErrTemplateRender(templateName, err)
	}
	return buf.Bytes(), nil
}

func (tr *templateRenderer) RenderRecipe(ctx context.Context, recipe *Recipe, options Options) (*ScaffoldStats, error) {
	stats := &ScaffoldStats{TotalFiles: len(recipe.Files)}
	startTime := time.Now()
	defer func() { stats.Duration = time.Since(startTime) }()

	templatesUsed := make(map[string]bool)
	for i, file := range recipe.Files {
		select {
		case <-ctx.Done():
			return stats, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "rendering cancelled")
		default:
		}
		templatesUsed[file.Template] = true
		renderCtx := tr.prepareRenderContext(recipe, file, options)
		if tr.fileExists(file.Path) && !options.Overwrite {
			stats.FilesSkipped++
			continue
		}
		content, err := tr.RenderTemplate(ctx, file.Template, renderCtx)
		if err != nil {
			stats.FilesFailed++
			if options.Dry {
				continue
			}
			return stats, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to render file").
				WithDetails("fileIndex", i).
				WithDetails("filePath", file.Path).
				WithDetails("template", file.Template)
		}
		if !options.Dry {
			if err := tr.writeFile(ctx, file.Path, content); err != nil {
				stats.FilesFailed++
				return stats, gerror.Wrap(err, gerror.ErrCodeIO, "failed to write file").
					WithDetails("filePath", file.Path)
			}
		}
		stats.FilesGenerated++
	}
	stats.TemplatesParsed = len(templatesUsed)
	return stats, nil
}

func (tr *templateRenderer) getTemplate(ctx context.Context, templateName, templatesDir string) (*template.Template, error) {
	if cached := tr.templateCache.Get(templateName); cached != nil {
		return cached, nil
	}
	templatePath := filepath.Join(templatesDir, templateName)
	content, err := fs.ReadFile(tr.templateFS, templatePath)
	if err != nil {
		return nil, ErrTemplateNotFound(templateName, templatesDir)
	}
	tmpl := template.New(templateName).Funcs(tr.funcMap)
	parsedTemplate, err := tmpl.Parse(string(content))
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeParsing, "failed to parse template").
			WithDetails("template", templateName)
	}
	tr.templateCache.Set(templateName, parsedTemplate)
	return parsedTemplate, nil
}

func (tr *templateRenderer) prepareRenderContext(recipe *Recipe, file FileEntry, options Options) RenderContext {
	vars := make(map[string]any)
	for k, v := range recipe.Vars {
		vars[k] = v
	}
	for k, v := range options.Vars {
		vars[k] = v
	}
	for k, v := range file.With {
		vars[k] = v
	}
	return RenderContext{Vars: vars, File: file, Recipe: recipe}
}

func (tr *templateRenderer) prepareTemplateData(context RenderContext) map[string]any {
	return map[string]any{
		"vars":   context.Vars,
		"with":   context.File.With,
		"file":   context.File,
		"recipe": context.Recipe,
	}
}

func (tr *templateRenderer) fileExists(path string) bool {
	_, err := tr.fileSystem.Stat(path)
	return err == nil
}

func (tr *templateRenderer) writeFile(ctx context.Context, path string, content []byte) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled before writing file")
	}
	dir := filepath.Dir(path)
	if err := tr.fileSystem.MkdirAll(dir, 0755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to create directory").
			WithDetails("directory", dir)
	}
	if err := tr.fileSystem.WriteFile(path, content, 0644); err != nil {
		return ErrFileWrite(path, err)
	}
	return nil
}

type TemplateCache struct {
	cache   map[string]*templateCacheEntry
	mu      sync.RWMutex
	maxSize int
	ttl     time.Duration
}

type templateCacheEntry struct {
	template  *template.Template
	timestamp time.Time
	hits      int64
}

func NewTemplateCache(maxSize int, ttl time.Duration) *TemplateCache {
	return &TemplateCache{cache: make(map[string]*templateCacheEntry), maxSize: maxSize, ttl: ttl}
}

func (tc *TemplateCache) Get(name string) *template.Template {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	entry, exists := tc.cache[name]
	if !exists {
		return nil
	}
	if time.Since(entry.timestamp) > tc.ttl {
		return nil
	}
	entry.hits++
	return entry.template
}

func (tc *TemplateCache) Set(name string, tmpl *template.Template) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	if len(tc.cache) >= tc.maxSize {
		tc.evictLRU()
	}
	tc.cache[name] = &templateCacheEntry{template: tmpl, timestamp: time.Now(), hits: 1}
}

func (tc *TemplateCache) Clear() {
	tc.mu.Lock()
	tc.cache = make(map[string]*templateCacheEntry)
	tc.mu.Unlock()
}

func (tc *TemplateCache) evictLRU() {
	var oldestName string
	oldestTime := time.Now()
	for name, entry := range tc.cache {
		if entry.timestamp.Before(oldestTime) {
			oldestTime = entry.timestamp
			oldestName = name
		}
	}
	if oldestName != "" {
		delete(tc.cache, oldestName)
	}
}

func getTemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"title": strings.Title,
		"trim":  strings.TrimSpace,

		"default": func(def, val any) any {
			switch v := val.(type) {
			case string:
				if strings.TrimSpace(v) == "" {
					return def
				}
				return v
			case nil:
				return def
			default:
				return val
			}
		},

		"join": func(list []string, sep string) string { return strings.Join(list, sep) },

		"isEmpty": func(v any) bool {
			switch v := v.(type) {
			case nil:
				return true
			case string:
				return strings.TrimSpace(v) == ""
			case []any:
				return len(v) == 0
			case map[string]any:
				return len(v) == 0
			default:
				return false
			}
		},
		"not": func(val bool) bool { return !val },

		"isString": func(val any) bool { _, ok := val.(string); return ok },
		"isMap":    func(val any) bool { _, ok := val.(map[string]any); return ok },
		"isList":   func(val any) bool { _, ok := val.([]any); return ok },

		"pathBase":  filepath.Base,
		"pathDir":   filepath.Dir,
		"pathExt":   filepath.Ext,
		"pathJoin":  filepath.Join,
		"pathClean": filepath.Clean,

		"now":     func() time.Time { return time.Now() },
		"date":    func(format string) string { return time.Now().Format(format) },
		"dateISO": func() string { return time.Now().Format(time.RFC3339) },

		"campaignHash": func(name string) string { return generateSimpleHash(name) },
		"quote":        func(s string) string { return `"` + s + `"` },
		"indent": func(spaces int, text string) string {
			indent := strings.Repeat(" ", spaces)
			lines := strings.Split(text, "\n")
			for i, line := range lines {
				if strings.TrimSpace(line) != "" {
					lines[i] = indent + line
				}
			}
			return strings.Join(lines, "\n")
		},
		"toYAML": func(v any) string { return convertToYAML(v) },
		"toJSON": func(v any) string { return convertToJSON(v) },
	}
}

func generateSimpleHash(input string) string {
	hasher := sha256.New()
	hasher.Write([]byte(input))
	hash := hex.EncodeToString(hasher.Sum(nil))
	return hash[:8]
}

func convertToYAML(v any) string {
	switch val := v.(type) {
	case nil:
		return "null"
	case bool:
		if val {
			return "true"
		}
		return "false"
	case int, int32, int64, float32, float64:
		return fmt.Sprintf("%v", val)
	case string:
		if strings.Contains(val, "\n") || strings.Contains(val, ":") || strings.Contains(val, "#") {
			return fmt.Sprintf("%q", val)
		}
		return val
	case []any:
		var items []string
		for _, item := range val {
			items = append(items, "- "+convertToYAML(item))
		}
		return strings.Join(items, "\n")
	case map[string]any:
		var items []string
		for key, value := range val {
			items = append(items, fmt.Sprintf("%s: %s", key, convertToYAML(value)))
		}
		return strings.Join(items, "\n")
	default:
		return fmt.Sprintf("%v", val)
	}
}

func convertToJSON(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(data)
}
