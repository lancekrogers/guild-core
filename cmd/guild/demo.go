package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lancekrogers/guild-core/pkg/config"
	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/spf13/cobra"
)

// demoCmd represents the demo command
var demoCmd = &cobra.Command{
	Use:   "demo",
	Short: "Run Guild framework demonstrations",
	Long: `Run various demonstrations of Guild framework capabilities.

These demos showcase Guild's multi-agent orchestration, task management,
and collaborative features in action.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDemo(cmd.Context())
	},
}

// demoCheckCmd represents the demo-check command
var demoCheckCmd = &cobra.Command{
	Use:   "demo-check",
	Short: "Check if Guild is properly configured for demos",
	Long: `Verify that Guild is properly initialized and configured
to run framework demonstrations.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDemoCheck(cmd.Context())
	},
}

func init() {
	rootCmd.AddCommand(demoCmd)
	rootCmd.AddCommand(demoCheckCmd)
}

func runDemo(ctx context.Context) error {
	// Check if we're in a Guild project
	if err := checkGuildProject(ctx); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeValidation, "not in a Guild project")
	}

	fmt.Println("🏰 Guild Framework Demo")
	fmt.Println("=====================")
	fmt.Println()
	fmt.Println("Available demos:")
	fmt.Println("1. Multi-agent coordination")
	fmt.Println("2. Task orchestration")
	fmt.Println("3. Knowledge management")
	fmt.Println()
	fmt.Println("Run 'guild demo [demo-name]' to start a specific demo")

	return nil
}

func runDemoCheck(ctx context.Context) error {
	// Check context first
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	fmt.Println("🏰 Guild Demo Check")
	fmt.Println("==================")
	fmt.Println()

	// Check for Guild project
	projectFound := false
	projectPath := ""

	// Check for .campaign structure (new format)
	campaignPath := filepath.Join(".", ".campaign")
	if info, err := os.Stat(campaignPath); err == nil && info.IsDir() {
		projectFound = true
		projectPath = campaignPath
		fmt.Println("✅ Found Guild campaign directory:", campaignPath)
	}

	// Legacy .guild workspace-level support removed
	// Only .campaign directories are supported at workspace level

	if !projectFound {
		fmt.Println("❌ Not in a Guild project")
		fmt.Println()
		fmt.Println("To initialize a new Guild project, run:")
		fmt.Println("  guild init")
		return gerror.New(gerror.ErrCodeNotFound, "not in a Guild project", nil)
	}

	// Try to load configuration
	cfg, err := config.LoadGuildConfig(ctx, ".")
	if err != nil {
		fmt.Printf("⚠️  Could not load Guild configuration: %v\n", err)
		fmt.Println()
		fmt.Println("Project structure found at:", projectPath)
		fmt.Println("But configuration may be invalid or incomplete.")
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to load configuration")
	}

	fmt.Printf("✅ Guild configuration loaded successfully\n")
	fmt.Printf("   Campaign: %s\n", cfg.Name)
	if len(cfg.Agents) > 0 {
		fmt.Printf("   Agents: %d configured\n", len(cfg.Agents))
	}

	fmt.Println()
	fmt.Println("🎉 Guild is ready for demos!")

	return nil
}

func checkGuildProject(ctx context.Context) error {
	// Check context
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	// Check for .campaign directory
	campaignExists := false

	if info, err := os.Stat(".campaign"); err == nil && info.IsDir() {
		campaignExists = true
	}

	if !campaignExists {
		return gerror.New(gerror.ErrCodeNotFound, "not in a Guild campaign - run 'guild init' to create one", nil)
	}

	return nil
}
