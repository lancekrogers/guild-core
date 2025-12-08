package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	scaffold "github.com/guild-framework/guild-core/internal/scaffoldlite"
	templ "github.com/guild-framework/guild-core/internal/scaffoldlite/templates"
	"github.com/guild-framework/guild-core/pkg/gerror"
)

// ExecuteInit executes the embedded template-driven initialization
func ExecuteInit(ctx context.Context, options *InitOptions) error {
	if options.ProjectName == "" {
		return gerror.New(gerror.ErrCodeValidation, "project name cannot be empty", nil)
	}
	if options.OutputDirectory == "" {
		return gerror.New(gerror.ErrCodeValidation, "output directory cannot be empty", nil)
	}

	templateFS, err := templ.GetEmbeddedTemplatesFS()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to access templates")
	}

	// Determine scaffold path within embedded FS
	scaffoldPath := filepath.Join(options.TemplateName, "scaffold.yaml")
	if options.TemplateName == "" {
		scaffoldPath = filepath.Join("campaign", "scaffold.yaml")
	}

	vars := map[string]any{}
	for k, v := range options.Variables {
		vars[k] = v
	}
	// Add common defaults
	if _, ok := vars["campaign_name"]; !ok {
		vars["campaign_name"] = options.ProjectName
	}
	if _, ok := vars["project_name"]; !ok {
		vars["project_name"] = options.ProjectName
	}
	vars["scaffold_version"] = "0.1.0"
	vars["runtime"] = map[string]any{"user": "guild-core", "go_version": "", "os": "", "arch": ""}

	sopts := scaffold.Options{TemplatesFS: templateFS, ScaffoldPath: scaffoldPath, Dest: options.OutputDirectory, Dry: options.DryRun, Overwrite: options.Force, Vars: vars}

	start := time.Now()
	stats, err := scaffold.ScaffoldFromFS(ctx, templateFS, scaffoldPath, sopts)
	if err != nil {
		return err
	}

	// Minimal output (avoid coupling to TUI/pretty printing)
	fmt.Println("✅ Scaffold Operation Complete!")
	fmt.Printf("   Duration: %v\n", time.Since(start))
	fmt.Printf("   Files generated: %d\n", stats.FilesGenerated)
	if stats.FilesSkipped > 0 {
		fmt.Printf("   Files skipped: %d\n", stats.FilesSkipped)
	}
	if stats.FilesFailed > 0 {
		fmt.Printf("   Files failed: %d\n", stats.FilesFailed)
	}
	return nil
}
