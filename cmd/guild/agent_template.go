// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/guild-ventures/guild-core/internal/setup"
	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// newAgentTemplateCmd creates the agent template subcommand
func newAgentTemplateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "template",
		Short: "Generate agent configurations from templates",
		Long: `Generate agent configuration files from lightweight templates.

This command provides a quick way to create agent configurations without
writing extensive YAML files. Templates include smart defaults and only
require essential fields.`,
	}

	cmd.AddCommand(
		newAgentTemplateListCmd(),
		newAgentTemplateGenerateCmd(),
		newAgentTemplateQuickCmd(),
	)

	return cmd
}

// newAgentTemplateListCmd lists available templates
func newAgentTemplateListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available agent templates",
		RunE: func(cmd *cobra.Command, args []string) error {
			generator := setup.NewAgentTemplateGenerator()
			templates := generator.ListTemplates()

			fmt.Println("Available Agent Templates:")
			fmt.Println("=========================")
			fmt.Println()

			// Group by provider
			byProvider := make(map[string][]string)
			for _, name := range templates {
				if template, exists := generator.GetTemplate(name); exists {
					provider := template.Provider
					if provider == "" {
						provider = "generic"
					}
					byProvider[provider] = append(byProvider[provider], name)
				}
			}

			// Display grouped
			for provider, names := range byProvider {
				fmt.Printf("%s:\n", strings.Title(provider))
				for _, name := range names {
					if template, exists := generator.GetTemplate(name); exists {
						fmt.Printf("  • %s - %s\n", name, template.Description)
					}
				}
				fmt.Println()
			}

			return nil
		},
	}
}

// newAgentTemplateGenerateCmd generates an agent from a template
func newAgentTemplateGenerateCmd() *cobra.Command {
	var (
		templateName string
		projectPath  string
		provider     string
		model        string
	)

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate an agent configuration from a template",
		Long: `Generate an agent configuration file from a template.

You can use a built-in template or create a custom one by specifying
the required fields.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			generator := setup.NewAgentTemplateGenerator()

			// Use built-in template if specified
			if templateName != "" {
				template, exists := generator.GetTemplate(templateName)
				if !exists {
					return gerror.Newf(gerror.ErrCodeNotFound, "template '%s' not found", templateName).
						WithComponent("agent-template").
						WithOperation("generate")
				}

				// Override provider/model if specified
				if provider != "" {
					template.Provider = provider
				}
				if model != "" {
					template.Model = model
				}

				if err := generator.GenerateAgentFile(cmd.Context(), projectPath, template); err != nil {
					return err
				}

				fmt.Printf("✓ Generated agent configuration: .guild/agents/%s.yml\n", strings.ToLower(template.ID))
				return nil
			}

			// Create custom template
			if len(args) < 1 {
				return gerror.New(gerror.ErrCodeValidation, "agent ID required", nil).
					WithComponent("agent-template").
					WithOperation("generate")
			}

			agentID := args[0]
			agentName := agentID
			if len(args) > 1 {
				agentName = args[1]
			}

			// Prompt for minimal fields
			fmt.Println("Creating custom agent template...")

			if provider == "" {
				return gerror.New(gerror.ErrCodeValidation, "provider required (use --provider flag)", nil).
					WithComponent("agent-template").
					WithOperation("generate")
			}
			if model == "" {
				return gerror.New(gerror.ErrCodeValidation, "model required (use --model flag)", nil).
					WithComponent("agent-template").
					WithOperation("generate")
			}

			template := generator.CreateCustomTemplate(
				agentID,
				agentName,
				"worker", // Default type
				provider,
				model,
				fmt.Sprintf("Custom %s agent", agentName),
				[]string{"general_tasks", "coding", "analysis"},
			)

			if err := generator.GenerateAgentFile(cmd.Context(), projectPath, template); err != nil {
				return err
			}

			fmt.Printf("✓ Generated agent configuration: .guild/agents/%s.yml\n", strings.ToLower(agentID))
			return nil
		},
	}

	cmd.Flags().StringVarP(&templateName, "template", "t", "", "Use a built-in template")
	cmd.Flags().StringVarP(&projectPath, "path", "p", ".", "Project path")
	cmd.Flags().StringVar(&provider, "provider", "", "Override provider")
	cmd.Flags().StringVar(&model, "model", "", "Override model")

	return cmd
}

// newAgentTemplateQuickCmd creates a minimal guild setup
func newAgentTemplateQuickCmd() *cobra.Command {
	var (
		projectPath  string
		provider     string
		managerModel string
		workerModel  string
	)

	cmd := &cobra.Command{
		Use:   "quick-setup",
		Short: "Quickly set up a manager and worker agent",
		Long: `Create a minimal guild setup with one manager and one worker agent.

This is the fastest way to get started with the Guild Framework.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if provider == "" {
				return gerror.New(gerror.ErrCodeValidation, "provider required", nil).
					WithComponent("agent-template").
					WithOperation("quick-setup")
			}
			if managerModel == "" {
				return gerror.New(gerror.ErrCodeValidation, "manager-model required", nil).
					WithComponent("agent-template").
					WithOperation("quick-setup")
			}
			if workerModel == "" {
				workerModel = managerModel // Use same model if not specified
			}

			generator := setup.NewAgentTemplateGenerator()
			if err := generator.QuickSetup(cmd.Context(), projectPath, provider, managerModel, workerModel); err != nil {
				return err
			}

			fmt.Println("✓ Quick setup complete!")
			fmt.Println()
			fmt.Println("Created:")
			fmt.Println("  • .guild/agents/manager.yml")
			fmt.Println("  • .guild/agents/worker-1.yml")
			fmt.Println()
			fmt.Println("Next steps:")
			fmt.Println("  1. Review the generated agent configurations")
			fmt.Println("  2. Run 'guild init' to complete guild setup")
			fmt.Println("  3. Start chatting with 'guild chat'")

			return nil
		},
	}

	cmd.Flags().StringVarP(&projectPath, "path", "p", ".", "Project path")
	cmd.Flags().StringVar(&provider, "provider", "", "Provider (e.g., openai, anthropic, ollama)")
	cmd.Flags().StringVar(&managerModel, "manager-model", "", "Model for manager agent")
	cmd.Flags().StringVar(&workerModel, "worker-model", "", "Model for worker agent (defaults to manager model)")

	return cmd
}
