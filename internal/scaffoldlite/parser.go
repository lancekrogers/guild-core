package scaffoldlite

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"gopkg.in/yaml.v3"
)

type Parser interface {
	ParseRecipe(ctx context.Context, fsys fs.FS, path string) (*Recipe, error)
	ValidateRecipe(ctx context.Context, recipe *Recipe) []ValidationError
	ParseWithOptions(ctx context.Context, data []byte, opts ParseOptions) (*Recipe, error)
}

type yamlParser struct {
	validator *semanticValidator
	options   ParseOptions
	cache     *ParserCache
}

func NewParser(options ParseOptions) (Parser, error) {
	validator := newSemanticValidator()
	cache := &ParserCache{cache: make(map[string]*cacheEntry), maxSize: 100, ttl: 5 * time.Minute}
	return &yamlParser{validator: validator, options: options, cache: cache}, nil
}

func (p *yamlParser) ParseRecipe(ctx context.Context, fsys fs.FS, path string) (*Recipe, error) {
	cacheKey := fmt.Sprintf("%s:%s", getFilesystemID(fsys), path)
	if cached, found := p.cache.Get(cacheKey); found {
		return cached, nil
	}
	data, err := p.readWithLimit(fsys, path)
	if err != nil {
		return nil, ErrFileRead(path, err)
	}
	recipe, err := p.parseWithTimeout(ctx, data, path)
	if err != nil {
		return nil, err
	}
	p.cache.Set(cacheKey, recipe)
	return recipe, nil
}

func (p *yamlParser) ValidateRecipe(ctx context.Context, recipe *Recipe) []ValidationError {
	return p.validator.Validate(ctx, recipe)
}

func (p *yamlParser) ParseWithOptions(ctx context.Context, data []byte, opts ParseOptions) (*Recipe, error) {
	tempParser := &yamlParser{validator: p.validator, options: opts, cache: p.cache}
	return tempParser.parseWithTimeout(ctx, data, "")
}

func (p *yamlParser) readWithLimit(fsys fs.FS, path string) ([]byte, error) {
	info, err := fs.Stat(fsys, path)
	if err != nil {
		return nil, err
	}
	if info.Size() > p.options.MaxFileSize {
		return nil, gerror.New(ErrCodeValidation, "file too large", nil).
			WithDetails("size", info.Size()).
			WithDetails("maxSize", p.options.MaxFileSize).
			WithDetails("path", path)
	}
	return fs.ReadFile(fsys, path)
}

func (p *yamlParser) parseWithTimeout(ctx context.Context, data []byte, filename string) (*Recipe, error) {
	parseCtx, cancel := context.WithTimeout(ctx, p.options.MaxParseTime)
	defer cancel()
	type parseResult struct {
		recipe *Recipe
		err    error
	}
	resultChan := make(chan parseResult, 1)
	go func() {
		recipe, err := p.parseYAML(data, filename)
		resultChan <- parseResult{recipe: recipe, err: err}
	}()
	select {
	case result := <-resultChan:
		if result.err != nil {
			return nil, result.err
		}
		if p.options.ValidateSemantics {
			if validationErrors := p.ValidateRecipe(parseCtx, result.recipe); len(validationErrors) > 0 {
				return nil, gerror.New(ErrCodeValidation, "recipe validation failed", nil).
					WithDetails("errors", validationErrors)
			}
		}
		return result.recipe, nil
	case <-parseCtx.Done():
		return nil, ErrTimeout("YAML parsing", p.options.MaxParseTime.String())
	}
}

func (p *yamlParser) parseYAML(data []byte, filename string) (*Recipe, error) {
	if isTreeFormat(data) {
		treeParser := NewTreeParser()
		recipe, err := treeParser.ParseTreeFormat(data)
		if err != nil {
			return nil, gerror.Wrap(err, ErrCodeYAMLParse, "failed to parse tree format").WithDetails("file", filename)
		}
		return recipe, nil
	}
	var recipe Recipe
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(p.options.StrictMode)
	if err := decoder.Decode(&recipe); err != nil {
		return nil, p.enhanceYAMLError(err, data, filename)
	}
	if err := p.validateRequiredFields(&recipe); err != nil {
		return nil, err
	}
	p.applyDefaults(&recipe)
	return &recipe, nil
}

func isTreeFormat(data []byte) bool {
	dataStr := string(data)
	hasTreeIndicators := strings.Contains(dataStr, "/:") && (strings.Contains(dataStr, "_files:") || strings.Contains(dataStr, "_empty:"))
	return hasTreeIndicators
}

// Parser cache
type ParserCache struct {
	cache   map[string]*cacheEntry
	mu      sync.RWMutex
	maxSize int
	ttl     time.Duration
}
type cacheEntry struct {
	recipe    *Recipe
	timestamp time.Time
}

func (pc *ParserCache) Get(key string) (*Recipe, bool) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	entry, ok := pc.cache[key]
	if !ok {
		return nil, false
	}
	if time.Since(entry.timestamp) > pc.ttl {
		return nil, false
	}
	return entry.recipe, true
}
func (pc *ParserCache) Set(key string, r *Recipe) {
	pc.mu.Lock()
	pc.cache[key] = &cacheEntry{recipe: r, timestamp: time.Now()}
	pc.mu.Unlock()
}

// Helpers adapted from original to keep minimal behavior
func getFilesystemID(fsys fs.FS) string { return fmt.Sprintf("%T", fsys) }

// YAML error enhancement (minimal placeholder)
func (p *yamlParser) enhanceYAMLError(err error, _ []byte, filename string) error {
	// Strip common yaml.v3 prefixes and add filename context
	msg := err.Error()
	msg = regexp.MustCompile(`(?i)line\s+\d+`).ReplaceAllString(msg, "")
	return gerror.Wrap(fmt.Errorf(strings.TrimSpace(msg)), ErrCodeYAMLParse, "failed to parse YAML").WithDetails("file", filename)
}

func (p *yamlParser) validateRequiredFields(recipe *Recipe) error {
	if recipe.Files == nil {
		return gerror.New(ErrCodeValidation, "missing files list in scaffold", nil)
	}
	return nil
}

func (p *yamlParser) applyDefaults(recipe *Recipe) {
	// Ensure templates_dir defaults to empty string (templates at root of FS)
	if recipe.TemplatesDir == "" {
		recipe.TemplatesDir = ""
	}
}
