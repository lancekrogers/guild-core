package scaffoldlite

import (
	"context"
	"io/fs"
	"time"

	"github.com/lancekrogers/guild-core/pkg/gerror"
)

// Engine provides the main scaffold operations interface
type Engine interface {
	LoadRecipeFS(ctx context.Context, fsys fs.FS, path string) (*Recipe, error)
	RenderFS(ctx context.Context, recipe *Recipe, options Options) (*ScaffoldStats, error)
	ValidateRecipe(ctx context.Context, recipe *Recipe) []ValidationError
	SetTemplateFS(fsys fs.FS)
	DryRun(ctx context.Context, recipe *Recipe, options Options) (*ScaffoldStats, error)
}

// ScaffoldEngine implements the Engine interface
type ScaffoldEngine struct {
	parser     Parser
	renderer   Renderer
	fileSystem FileSystem
}

// NewEngine creates a new scaffold engine
func NewEngine(templateFS fs.FS, fileSystem FileSystem) (Engine, error) {
	parser, err := NewParser(DefaultParseOptions)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create parser")
	}

	renderer := NewRenderer(templateFS, fileSystem)

	if validator, ok := parser.(*yamlParser); ok {
		validator.validator.SetTemplateFS(templateFS)
	}

	return &ScaffoldEngine{
		parser:     parser,
		renderer:   renderer,
		fileSystem: fileSystem,
	}, nil
}

func (se *ScaffoldEngine) LoadRecipeFS(ctx context.Context, fsys fs.FS, path string) (*Recipe, error) {
	return se.parser.ParseRecipe(ctx, fsys, path)
}

func (se *ScaffoldEngine) RenderFS(ctx context.Context, recipe *Recipe, options Options) (*ScaffoldStats, error) {
	if validationErrors := se.ValidateRecipe(ctx, recipe); len(validationErrors) > 0 {
		return nil, gerror.New(ErrCodeValidation, "recipe validation failed", nil).
			WithDetails("errors", validationErrors)
	}
	if err := se.fileSystem.MkdirAll(".", 0o755); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeIO, "failed to create destination directory").
			WithDetails("dest", options.Dest)
	}
	if options.TemplatesFS != nil {
		se.renderer.SetTemplateFS(options.TemplatesFS)
	}
	return se.renderer.RenderRecipe(ctx, recipe, options)
}

func (se *ScaffoldEngine) ValidateRecipe(ctx context.Context, recipe *Recipe) []ValidationError {
	return se.parser.ValidateRecipe(ctx, recipe)
}

func (se *ScaffoldEngine) SetTemplateFS(fsys fs.FS) {
	se.renderer.SetTemplateFS(fsys)
	if parser, ok := se.parser.(*yamlParser); ok {
		parser.validator.SetTemplateFS(fsys)
	}
}

func (se *ScaffoldEngine) DryRun(ctx context.Context, recipe *Recipe, options Options) (*ScaffoldStats, error) {
	dryOptions := options
	dryOptions.Dry = true
	return se.RenderFS(ctx, recipe, dryOptions)
}

func ScaffoldFromFS(ctx context.Context, templateFS fs.FS, scaffoldPath string, options Options) (*ScaffoldStats, error) {
	fileSystem, err := NewOSFileSystem(options.Dest)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create filesystem")
	}
	engine, err := NewEngine(templateFS, fileSystem)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create scaffold engine")
	}
	recipe, err := engine.LoadRecipeFS(ctx, templateFS, scaffoldPath)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeParsing, "failed to load recipe")
	}
	options.TemplatesFS = templateFS
	return engine.RenderFS(ctx, recipe, options)
}

func DryRunFromFS(ctx context.Context, templateFS fs.FS, scaffoldPath string, options Options) (*ScaffoldStats, error) {
	options.Dry = true
	return ScaffoldFromFS(ctx, templateFS, scaffoldPath, options)
}

func GetScaffoldInfo(ctx context.Context, templateFS fs.FS, scaffoldPath string) (*ScaffoldInfo, error) {
	start := time.Now()
	memFS := NewMemoryFileSystem()
	engine, err := NewEngine(templateFS, memFS)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create scaffold engine")
	}
	recipe, err := engine.LoadRecipeFS(ctx, templateFS, scaffoldPath)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeParsing, "failed to load recipe")
	}
	loadTime := time.Since(start)

	templateSet := make(map[string]bool)
	for _, file := range recipe.Files {
		templateSet[file.Template] = true
	}
	templates := make([]string, 0, len(templateSet))
	for template := range templateSet {
		templates = append(templates, template)
	}
	var estimatedSize int64
	for template := range templateSet {
		if data, err := fs.ReadFile(templateFS, template); err == nil {
			estimatedSize += int64(len(data))
		}
	}
	return &ScaffoldInfo{
		ScaffoldVersion: recipe.ScaffoldVersion,
		TemplatesDir:    recipe.TemplatesDir,
		FileCount:       len(recipe.Files),
		Variables:       recipe.Vars,
		Templates:       templates,
		EstimatedSize:   estimatedSize,
		LoadTime:        loadTime,
	}, nil
}

type ScaffoldInfo struct {
	ScaffoldVersion string         `json:"scaffold_version"`
	TemplatesDir    string         `json:"templates_dir"`
	FileCount       int            `json:"file_count"`
	Variables       map[string]any `json:"variables"`
	Templates       []string       `json:"templates"`
	EstimatedSize   int64          `json:"estimated_size_bytes"`
	LoadTime        time.Duration  `json:"load_time"`
}
