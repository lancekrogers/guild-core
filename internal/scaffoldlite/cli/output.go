package cli

import (
    "context"
    "fmt"
    "os"
    "path/filepath"
    "strings"
)

// DetectTemplateFromContext performs simple on-disk checks to choose a template
func DetectTemplateFromContext(_ context.Context, outputDir string) string {
    detectors := []struct{ name, pattern, template string }{
        {"existing_campaign", ".campaign/campaign.yaml", "campaign"},
        {"existing_guild_core", "pkg/agent/interface.go", "guild_core_extension"},
        {"go_module", "go.mod", "single_agent"},
        {"typescript_project", "package.json", "single_agent"},
    }
    for _, d := range detectors {
        path := filepath.Join(outputDir, d.pattern)
        if _, err := os.Stat(path); err == nil {
            fmt.Printf("🔍 Detected %s, using %s template\n", d.name, d.template)
            return d.template
        }
    }
    fmt.Println("📝 No existing project detected, using campaign template")
    return "campaign"
}

// ListTemplates prints available embedded templates (static list)
func ListTemplates(ctx context.Context) error {
    _ = ctx
    templates := []struct{ name, desc, use, cat string }{
        {"campaign", "Complete campaign workspace with guild configuration", "New multi-project workspace with coordinated guilds", "workspace"},
        {"guild_core_extension", "Extension to existing guild-core repository", "Adding new features or capabilities to guild-core", "extension"},
        {"single_agent", "Simple single-agent project", "Rapid prototyping or simple automation tasks", "agent"},
    }
    fmt.Println("📋 Available Templates:\n")
    for _, t := range templates {
        fmt.Printf("🎯 %s\n   %s\n   Use case: %s\n   Category: %s\n\n", t.name, t.desc, t.use, t.cat)
    }
    fmt.Println("💡 Use --template <name> to select a specific template")
    fmt.Println("💡 Use --interactive for guided template selection")
    _ = strings.Builder{}
    return nil
}

