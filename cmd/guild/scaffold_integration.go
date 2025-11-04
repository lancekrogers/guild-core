// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/guild-framework/guild-scaffold/pkg/scaffold/cli"
)

// ScaffoldIntegration provides scaffold integration with guild init
type ScaffoldIntegration struct {
	enabled     bool
	interactive bool
	verbose     bool
}

// NewScaffoldIntegration creates a new scaffold integration
func NewScaffoldIntegration() *ScaffoldIntegration {
	return &ScaffoldIntegration{
		enabled:     getEnvBool("GUILD_SCAFFOLD_ENABLED", true),
		interactive: getEnvBool("GUILD_SCAFFOLD_INTERACTIVE", true),
		verbose:     getEnvBool("GUILD_SCAFFOLD_VERBOSE", false),
	}
}

// IsEnabled returns whether scaffold integration is enabled
func (si *ScaffoldIntegration) IsEnabled() bool {
	return si.enabled
}

// ExecuteScaffoldInit executes scaffold-based initialization
func (si *ScaffoldIntegration) ExecuteScaffoldInit(ctx context.Context, cmd *cobra.Command, args []string) error {
	// Parse project name
	projectName := "guild-project"
	if len(args) > 0 {
		projectName = args[0]
	}
	
	// Get current directory
	outputDir, err := os.Getwd()
	if err != nil {
		return gerror.Wrap(err, "failed to get current directory")
	}
	
	// Check for output directory flag from original command
	if outputFlag := cmd.Flags().Lookup("output"); outputFlag != nil {
		if flagValue := outputFlag.Value.String(); flagValue != "" && flagValue != "." {
			outputDir, err = filepath.Abs(flagValue)
			if err != nil {
				return gerror.Wrap(err, "invalid output directory").
					WithField("path", flagValue)
			}
		}
	}
	
	// Check for template flag
	templateName := ""
	if templateFlag := cmd.Flags().Lookup("template"); templateFlag != nil {
		templateName = templateFlag.Value.String()
	}
	
	// Check for dry-run flag
	dryRun := false
	if dryRunFlag := cmd.Flags().Lookup("dry-run"); dryRunFlag != nil {
		if flag, ok := dryRunFlag.Value.(*boolValue); ok {
			dryRun = flag.value
		}
	}
	
	// Check for force flag
	force := false
	if forceFlag := cmd.Flags().Lookup("force"); forceFlag != nil {
		if flag, ok := forceFlag.Value.(*boolValue); ok {
			force = flag.value
		}
	}
	
	// Check for interactive flag
	interactive := si.interactive
	if interactiveFlag := cmd.Flags().Lookup("interactive"); interactiveFlag != nil {
		if flag, ok := interactiveFlag.Value.(*boolValue); ok {
			interactive = flag.value
		}
	}
	
	// Parse variables from flags
	variables := make(map[string]interface{})
	if varsFlag := cmd.Flags().Lookup("var"); varsFlag != nil {
		if varSlice, ok := varsFlag.Value.(*stringSliceValue); ok {
			parsedVars, err := parseVariables(varSlice.value)
			if err != nil {
				return gerror.Wrap(err, "failed to parse variables")
			}
			variables = parsedVars
		}
	}
	
	// Auto-detect template if not specified
	if templateName == "" {
		templateName = cli.DetectTemplateFromContext(ctx, outputDir)
	}
	
	// Create CLI options
	options := &cli.InitOptions{
		ProjectName:     projectName,
		TemplateName:    templateName,
		OutputDirectory: outputDir,
		Variables:       variables,
		DryRun:          dryRun,
		Force:           force,
		Interactive:     interactive,
		Verbose:         si.verbose,
	}
	
	// Show transition message
	fmt.Println("🚀 Using enhanced scaffold-based initialization")
	if si.verbose {
		fmt.Printf("   Template: %s\n", templateName)
		fmt.Printf("   Output: %s\n", outputDir)
		fmt.Printf("   Interactive: %v\n", interactive)
	}
	fmt.Println()
	
	// Execute scaffolding
	return cli.ExecuteInit(ctx, options)
}

// AddScaffoldFlags adds scaffold-specific flags to a command
func (si *ScaffoldIntegration) AddScaffoldFlags(cmd *cobra.Command) {
	if !si.enabled {
		return
	}
	
	cmd.Flags().String("template", "", "Template to use for initialization")
	cmd.Flags().Bool("dry-run", false, "Preview what would be created without creating files")
	cmd.Flags().Bool("force", false, "Overwrite existing files")
	cmd.Flags().StringSlice("var", nil, "Set template variables (format: key=value)")
	cmd.Flags().Bool("interactive", si.interactive, "Interactive mode with prompts")
	cmd.Flags().String("provider", "", "Default LLM provider for agents")
	cmd.Flags().String("model", "", "Default model for agents")
	
	// Add scaffold-specific commands if needed
	if listTemplatesCmd := si.createListTemplatesCmd(); listTemplatesCmd != nil {
		cmd.AddCommand(listTemplatesCmd)
	}
}

// createListTemplatesCmd creates a list-templates subcommand
func (si *ScaffoldIntegration) createListTemplatesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-templates",
		Short: "List available scaffold templates",
		Long:  `List all available templates that can be used with guild init.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			return cli.ListTemplates(ctx)
		},
	}
}

// ShouldUseScaffold determines if scaffold should be used instead of legacy init
func (si *ScaffoldIntegration) ShouldUseScaffold(ctx context.Context, args []string) bool {
	if !si.enabled {
		return false
	}
	
	// Check for scaffold-specific flags
	if len(args) > 0 {
		for _, arg := range args {
			if strings.HasPrefix(arg, "--template") ||
			   strings.HasPrefix(arg, "--dry-run") ||
			   strings.HasPrefix(arg, "--var") ||
			   strings.HasPrefix(arg, "--list-templates") {
				return true
			}
		}
	}
	
	// Default to scaffold for new installations
	return true
}

// parseVariables parses key=value variable assignments
func parseVariables(varStrings []string) (map[string]interface{}, error) {
	variables := make(map[string]interface{})
	
	for _, varStr := range varStrings {
		parts := strings.SplitN(varStr, "=", 2)
		if len(parts) != 2 {
			return nil, gerror.New("invalid variable format").
				WithField("variable", varStr).
				WithField("expected_format", "key=value")
		}
		
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		
		if key == "" {
			return nil, gerror.New("variable key cannot be empty").
				WithField("variable", varStr)
		}
		
		// Attempt to parse as different types
		variables[key] = parseVariableValue(value)
	}
	
	return variables, nil
}

// parseVariableValue attempts to parse a string value as appropriate type
func parseVariableValue(value string) interface{} {
	// Try boolean
	if value == "true" {
		return true
	}
	if value == "false" {
		return false
	}
	
	// Try integer
	if intVal, err := strconv.Atoi(value); err == nil {
		return intVal
	}
	
	// Try float
	if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
		return floatVal
	}
	
	// Return as string
	return value
}

// getEnvBool gets a boolean value from environment with default
func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	
	switch strings.ToLower(value) {
	case "true", "1", "yes", "on", "enabled":
		return true
	case "false", "0", "no", "off", "disabled":
		return false
	default:
		return defaultValue
	}
}

// Helper types to work with cobra flags
type boolValue struct {
	value bool
}

func (b *boolValue) String() string {
	return strconv.FormatBool(b.value)
}

func (b *boolValue) Set(val string) error {
	v, err := strconv.ParseBool(val)
	if err != nil {
		return err
	}
	b.value = v
	return nil
}

func (b *boolValue) Type() string {
	return "bool"
}

type stringSliceValue struct {
	value []string
}

func (s *stringSliceValue) String() string {
	return strings.Join(s.value, ",")
}

func (s *stringSliceValue) Set(val string) error {
	s.value = append(s.value, val)
	return nil
}

func (s *stringSliceValue) Type() string {
	return "stringSlice"
}