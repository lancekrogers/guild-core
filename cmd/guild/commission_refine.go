// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/guild-ventures/guild-core/pkg/agent/manager"
	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/kanban"
	"github.com/guild-ventures/guild-core/pkg/paths"
	"github.com/guild-ventures/guild-core/pkg/project"
	"github.com/guild-ventures/guild-core/pkg/prompts/layered"
	promptcontext "github.com/guild-ventures/guild-core/pkg/prompts/layered/context"
	"github.com/guild-ventures/guild-core/pkg/providers"
	"github.com/guild-ventures/guild-core/pkg/registry"
)

var (
	// Refinement specific flags
	domainFlag      string // Domain type (web-app, cli-tool, library, microservice)
	outputDirFlag   string // Output directory for refined objectives
	interactiveFlag bool   // Interactive mode for reviewing refinements
	createTasksFlag bool   // Create kanban tasks from refined content
)

// layeredManagerAdapter is a simple adapter for the layered.Manager interface
type layeredManagerAdapter struct{}

// Implement the layered.Manager interface methods
func (a *layeredManagerAdapter) GetSystemPrompt(ctx context.Context, role, domain string) (string, error) {
	// Return a default Guild Master prompt
	return `You are a Guild Master responsible for refining commissions into detailed implementation plans.
Your role is to break down high-level objectives into specific, actionable tasks that can be assigned to specialized artisans.`, nil
}

func (a *layeredManagerAdapter) GetTemplate(ctx context.Context, templateName string) (string, error) {
	return "", gerror.New(gerror.ErrCodeNotFound, "templates not supported in adapter", nil)
}

func (a *layeredManagerAdapter) FormatContext(ctx context.Context, context layered.Context) (string, error) {
	return "Context: Processing commission refinement", nil
}

func (a *layeredManagerAdapter) ListRoles(ctx context.Context) ([]string, error) {
	return []string{"manager", "worker", "specialist"}, nil
}

func (a *layeredManagerAdapter) ListDomains(ctx context.Context, role string) ([]string, error) {
	return []string{"web-app", "cli-tool", "library", "microservice"}, nil
}

// Implement LayeredManager specific methods
func (a *layeredManagerAdapter) BuildLayeredPrompt(ctx context.Context, artisanID, sessionID string, turnCtx layered.TurnContext) (*layered.LayeredPrompt, error) {
	// Simple implementation
	return &layered.LayeredPrompt{
		Compiled: a.getDefaultPrompt(),
	}, nil
}

func (a *layeredManagerAdapter) GetPromptLayer(ctx context.Context, layer layered.PromptLayer, artisanID, sessionID string) (*layered.SystemPrompt, error) {
	return nil, gerror.New(gerror.ErrCodeNotFound, "layer not found", nil)
}

func (a *layeredManagerAdapter) SetPromptLayer(ctx context.Context, prompt layered.SystemPrompt) error {
	return nil // No-op for adapter
}

func (a *layeredManagerAdapter) DeletePromptLayer(ctx context.Context, layer layered.PromptLayer, artisanID, sessionID string) error {
	return nil // No-op for adapter
}

func (a *layeredManagerAdapter) ListPromptLayers(ctx context.Context, artisanID, sessionID string) ([]layered.SystemPrompt, error) {
	return []layered.SystemPrompt{}, nil
}

func (a *layeredManagerAdapter) InvalidateCache(ctx context.Context, artisanID, sessionID string) error {
	return nil // No-op for adapter
}

func (a *layeredManagerAdapter) getDefaultPrompt() string {
	return `You are a Guild Master responsible for refining commissions into detailed implementation plans.
Your role is to break down high-level objectives into specific, actionable tasks that can be assigned to specialized artisans.`
}

// commissionRefineCmd refines a commission into a hierarchical objective structure
var commissionRefineCmd = &cobra.Command{
	Use:   "refine [commission description or file]",
	Short: "Refine a commission into detailed implementation plans",
	Long: `Refine a high-level commission into a hierarchical objective structure.

The Guild Master will analyze your commission and create:
- Detailed architecture and design documents
- Hierarchical task breakdowns
- Implementation specifications
- Task assignments for artisan agents

Examples:
  guild commission refine "Build a REST API for user management"
  guild commission refine commission.md --domain web-app
  guild commission refine "Create a CLI tool for file processing" --output ./commissions`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// Get commission content
		commissionContent, err := getCommissionContent(args)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInvalidInput, "failed to get commission content").
				WithComponent("cli").
				WithOperation("commission.refine")
		}

		// Execute refinement
		return executeRefinement(ctx, commissionContent)
	},
}

func init() {
	// Add refinement subcommand to commission
	commissionCmd.AddCommand(commissionRefineCmd)

	// Add refinement flags
	commissionRefineCmd.Flags().StringVar(&domainFlag, "domain", "web-app", "Project domain (web-app, cli-tool, library, microservice)")
	commissionRefineCmd.Flags().StringVar(&outputDirFlag, "output", "", "Output directory for refined commissions (default: .campaign/commissions/refined)")
	commissionRefineCmd.Flags().BoolVar(&interactiveFlag, "interactive", false, "Interactive mode for reviewing refinements")
	commissionRefineCmd.Flags().BoolVar(&createTasksFlag, "create-tasks", true, "Create kanban tasks from refined content")
}

// getCommissionContent retrieves commission content from args or file
func getCommissionContent(args []string) (string, error) {
	input := strings.Join(args, " ")

	// Check if input is a file path
	if strings.HasSuffix(input, ".md") || strings.HasSuffix(input, ".txt") {
		content, err := os.ReadFile(input)
		if err != nil {
			return "", gerror.Wrap(err, gerror.ErrCodeStorage, "failed to read commission file").
				WithComponent("cli").
				WithOperation("commission.refine").
				WithDetails("file", input)
		}
		return string(content), nil
	}

	return input, nil
}

// executeRefinement performs the commission refinement using GuildMasterRefiner
func executeRefinement(ctx context.Context, commissionContent string) error {
	fmt.Printf("🏰 Guild Master analyzing commission...\n\n")

	// Get project context
	projCtx, err := project.GetContext()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get project context").
			WithComponent("cli").
			WithOperation("commission.refine")
	}

	// Load guild configuration
	guildConfig, err := config.LoadGuildConfig(ctx, projCtx.GetRootPath())
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to load guild config").
			WithComponent("cli").
			WithOperation("commission.refine")
	}

	// Setup components for refinement
	refiner, err := setupRefiner(ctx, projCtx, guildConfig)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to setup refiner").
			WithComponent("cli").
			WithOperation("commission.refine")
	}

	// Determine output directory
	outputDir := outputDirFlag
	if outputDir == "" {
		outputDir = filepath.Join(projCtx.GetRootPath(), paths.DefaultCampaignDir, "commissions", "refined")
	}

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create output directory").
			WithComponent("cli").
			WithOperation("commission.refine").
			WithDetails("path", outputDir)
	}

	fmt.Printf("📝 Commission Details:\n")
	fmt.Printf("   Domain: %s\n", domainFlag)
	fmt.Printf("   Output: %s\n", outputDir)
	fmt.Printf("\n")

	// Show commission preview
	preview := commissionContent
	if len(preview) > 200 {
		preview = preview[:200] + "..."
	}
	fmt.Printf("Commission:\n%s\n\n", preview)

	fmt.Printf("🧠 Refining commission into implementation plans...\n\n")

	// Execute refinement
	refinedContent, err := refiner.RefineCommissionSimple(ctx, commissionContent, domainFlag)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeAgent, "refinement failed").
			WithComponent("cli").
			WithOperation("commission.refine").
			WithDetails("domain", domainFlag)
	}

	// Parse the refined content into file structures
	files, err := parseRefinedContent(refinedContent)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to parse refined content").
			WithComponent("cli").
			WithOperation("commission.refine")
	}

	fmt.Printf("✅ Refinement complete! Generated %d files\n\n", len(files))

	// Create kanban tasks if requested
	if createTasksFlag {
		tasks, err := createKanbanTasks(ctx, files, commissionContent, projCtx)
		if err != nil {
			fmt.Printf("⚠️  Warning: Failed to create kanban tasks: %v\n", err)
			// Continue execution - task creation failure shouldn't stop the refinement
		} else if len(tasks) > 0 {
			fmt.Printf("📋 Created %d kanban tasks from refined content\n\n", len(tasks))
		}
	}

	// Interactive review if requested
	if interactiveFlag {
		if err := interactiveReview(files); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "interactive review failed").
				WithComponent("cli").
				WithOperation("commission.refine")
		}
	}

	// Write files to output directory
	fmt.Printf("💾 Writing refined commissions to %s\n", outputDir)
	for _, file := range files {
		filePath := filepath.Join(outputDir, file.Path)

		// Create directory if needed
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create directory").
				WithComponent("cli").
				WithOperation("commission.refine").
				WithDetails("dir", dir)
		}

		// Write file
		if err := os.WriteFile(filePath, []byte(file.Content), 0644); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write file").
				WithComponent("cli").
				WithOperation("commission.refine").
				WithDetails("path", filePath)
		}

		fmt.Printf("   📄 %s\n", file.Path)
	}

	fmt.Printf("\n✨ Commission refined successfully!\n")
	fmt.Printf("Next steps:\n")
	fmt.Printf("1. Review the generated files in %s\n", outputDir)
	fmt.Printf("2. Use 'guild commission create %s' to create trackable commission\n", outputDir)
	fmt.Printf("3. Use 'guild commission assign' to assign tasks to artisans\n")

	return nil
}

// setupRefiner creates and configures the GuildMasterRefiner
func setupRefiner(ctx context.Context, projCtx *project.Context, guildConfig *config.GuildConfig) (*manager.GuildMasterRefiner, error) {
	// Setup data directory
	dataDir := filepath.Join(projCtx.GetRootPath(), paths.DefaultCampaignDir)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create data directory").
			WithComponent("cli").
			WithOperation("commission.refine.setupRefiner").
			WithDetails("path", dataDir)
	}

	// Initialize component registry with SQLite storage
	reg := registry.NewComponentRegistry()
	if err := reg.Initialize(ctx, registry.Config{}); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize registry").
			WithComponent("cli").
			WithOperation("commission.refine.setupRefiner")
	}

	// Create formatter
	_, err := promptcontext.NewXMLFormatter()
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create formatter").
			WithComponent("cli").
			WithOperation("commission.refine.setupRefiner")
	}

	// Refinement prompts are now loaded from the layered prompt system
	// if err := registerRefinementPrompts(promptRegistry); err != nil {
	// 	return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register prompts").
	// 		WithComponent("cli").
	// 		WithOperation("commission.refine.setupRefiner")
	// }

	// Create layered prompt manager
	// For now, use a simple adapter since we don't have the full layered manager setup
	layeredPromptManager := &layeredManagerAdapter{}

	// Get manager agent configuration
	managerID := getManagerAgentID(guildConfig)
	var managerConfig *config.AgentConfig
	for _, agent := range guildConfig.Agents {
		if agent.ID == managerID {
			managerConfig = &agent
			break
		}
	}

	if managerConfig == nil {
		return nil, gerror.New(gerror.ErrCodeAgentNotFound, "manager agent configuration not found", nil).
			WithComponent("cli").
			WithOperation("commission.refine.setupRefiner").
			WithDetails("agent_id", managerID)
	}

	// Create provider factory v2
	providerFactory := providers.NewFactoryV2()

	// Map provider name to type
	var providerType providers.ProviderType
	switch managerConfig.Provider {
	case "openai":
		providerType = providers.ProviderOpenAI
	case "anthropic":
		providerType = providers.ProviderAnthropic
	case "ollama":
		providerType = providers.ProviderOllama
	case "deepseek":
		providerType = providers.ProviderDeepSeek
	case "deepinfra":
		providerType = providers.ProviderDeepInfra
	case "ora":
		providerType = providers.ProviderOra
	default:
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "unsupported provider", nil).
			WithComponent("cli").
			WithOperation("commission.refine.setupRefiner").
			WithDetails("provider", managerConfig.Provider)
	}

	// Get API key
	apiKey := guildConfig.GetProviderAPIKey(managerConfig.Provider)
	if apiKey == "" && requiresAPIKey(managerConfig.Provider) {
		return nil, gerror.New(gerror.ErrCodeProviderAuth, "API key required", nil).
			WithComponent("cli").
			WithOperation("commission.refine.setupRefiner").
			WithDetails("provider", managerConfig.Provider).
			WithDetails("env_var", getEnvVarName(managerConfig.Provider))
	}

	// Create AI provider
	aiProvider, err := providerFactory.CreateAIProvider(providerType, apiKey)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeProvider, "failed to create AI provider").
			WithComponent("cli").
			WithOperation("commission.refine.setupRefiner").
			WithDetails("provider", managerConfig.Provider)
	}

	// Create artisan client adapter
	artisanClient := manager.NewGuildArtisanClient(aiProvider, managerConfig.Model)

	// Create GuildMasterRefiner
	refiner := manager.NewGuildMasterRefiner(
		artisanClient,
		layeredPromptManager,
		nil, // Parser will be simple for now
		nil, // Validator not needed for CLI demo
	)

	return refiner, nil
}

// registerRefinementPrompts registers all refinement prompts
// Commented out - refinement prompts are now loaded from the layered prompt system
/*
func registerRefinementPrompts(registry *prompts.PromptRegistry) error {
	// Register base manager refinement prompt
	if err := registry.RegisterPrompt("manager", "web-app", commission.ManagerRefinementPrompt + commission.WebAppDomainPrompt); err != nil {
		return err
	}

	if err := registry.RegisterPrompt("manager", "cli-tool", commission.ManagerRefinementPrompt + commission.CLIToolDomainPrompt); err != nil {
		return err
	}

	if err := registry.RegisterPrompt("manager", "library", commission.ManagerRefinementPrompt + commission.LibraryDomainPrompt); err != nil {
		return err
	}

	if err := registry.RegisterPrompt("manager", "microservice", commission.ManagerRefinementPrompt + commission.MicroserviceDomainPrompt); err != nil {
		return err
	}

	return nil
}
*/

// RefinedFile represents a file in the refined objective structure
type RefinedFile struct {
	Path    string
	Content string
}

// parseRefinedContent parses the LLM response into file structures
func parseRefinedContent(content string) ([]RefinedFile, error) {
	var files []RefinedFile

	// Split content by "## File:" markers
	sections := strings.Split(content, "## File:")

	for i := 1; i < len(sections); i++ { // Skip first empty section
		section := strings.TrimSpace(sections[i])
		if section == "" {
			continue
		}

		// Extract file path from first line
		lines := strings.Split(section, "\n")
		if len(lines) < 2 {
			continue
		}

		filePath := strings.TrimSpace(lines[0])

		// Get content (everything after first line)
		fileContent := strings.Join(lines[1:], "\n")
		fileContent = strings.TrimSpace(fileContent)

		files = append(files, RefinedFile{
			Path:    filePath,
			Content: fileContent,
		})
	}

	// If no files parsed using the marker, treat entire content as README.md
	if len(files) == 0 {
		files = append(files, RefinedFile{
			Path:    "README.md",
			Content: content,
		})
	}

	return files, nil
}

// createKanbanTasks creates kanban tasks from refined content
func createKanbanTasks(ctx context.Context, files []RefinedFile, commissionContent string, projCtx *project.Context) ([]*kanban.Task, error) {
	// Initialize registry for kanban operations
	reg := registry.NewComponentRegistry()
	if err := reg.Initialize(ctx, registry.Config{}); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize registry for kanban").
			WithComponent("cli").
			WithOperation("createKanbanTasks")
	}

	// Create or get the commission board
	boardName := fmt.Sprintf("Commission: %s", extractCommissionTitle(commissionContent))
	board, err := getOrCreateBoard(ctx, reg, boardName, commissionContent)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create commission board").
			WithComponent("cli").
			WithOperation("createKanbanTasks")
	}

	var createdTasks []*kanban.Task

	// Extract tasks from each refined file
	for _, file := range files {
		tasks := extractTasksFromContent(file.Content, file.Path)

		// Create kanban tasks for each extracted task
		for _, taskInfo := range tasks {
			task, err := board.CreateTask(ctx, taskInfo.Title, taskInfo.Description)
			if err != nil {
				fmt.Printf("⚠️  Warning: Failed to create task '%s': %v\n", taskInfo.Title, err)
				continue
			}

			// Add metadata about the source file
			if task.Metadata == nil {
				task.Metadata = make(map[string]string)
			}
			task.Metadata["source_file"] = file.Path
			task.Metadata["commission_type"] = domainFlag
			task.Metadata["created_from"] = "commission_refinement"

			createdTasks = append(createdTasks, task)
			fmt.Printf("   📝 Created task: %s\n", taskInfo.Title)
		}
	}

	return createdTasks, nil
}

// TaskInfo represents a task extracted from refined content
type TaskInfo struct {
	Title       string
	Description string
	Priority    string
	AgentType   string
}

// extractCommissionTitle extracts a title from the commission content
func extractCommissionTitle(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			// Take first non-empty, non-header line as title
			if len(line) > 50 {
				return line[:50] + "..."
			}
			return line
		}
	}
	return "Refined Commission"
}

// getOrCreateBoard gets an existing board or creates a new one for the commission
func getOrCreateBoard(ctx context.Context, reg registry.ComponentRegistry, boardName, commissionContent string) (*kanban.Board, error) {
	// Create adapter to bridge the interface gap (reuse existing adapter from campaign.go)
	kanbanReg := &kanbanComponentRegistry{componentReg: reg}

	// Create a new board for this commission using the proper constructor
	description := fmt.Sprintf("Kanban board for commission refinement in %s domain", domainFlag)

	board, err := kanban.NewBoardWithRegistry(ctx, kanbanReg, boardName, description)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create kanban board").
			WithComponent("cli").
			WithOperation("getOrCreateBoard")
	}

	// Add metadata about the commission
	if board.Metadata == nil {
		board.Metadata = make(map[string]string)
	}
	board.Metadata["commission_type"] = domainFlag
	board.Metadata["created_from"] = "commission_refinement"

	return board, nil
}

// extractTasksFromContent extracts task information from refined file content
func extractTasksFromContent(content, filePath string) []TaskInfo {
	var tasks []TaskInfo

	lines := strings.Split(content, "\n")

	// Look for task-like patterns in the content
	for i, line := range lines {
		line = strings.TrimSpace(line)

		// Look for markdown task lists (- [ ] Task name)
		if strings.HasPrefix(line, "- [ ]") || strings.HasPrefix(line, "* [ ]") {
			taskTitle := strings.TrimSpace(line[5:]) // Remove "- [ ]" prefix
			description := extractTaskDescription(lines, i+1)

			tasks = append(tasks, TaskInfo{
				Title:       taskTitle,
				Description: description,
				Priority:    "medium",
				AgentType:   inferAgentType(taskTitle, filePath),
			})
		}

		// Look for numbered task lists (1. Task name)
		if matched := strings.HasPrefix(line, "1.") || strings.HasPrefix(line, "2.") ||
			strings.HasPrefix(line, "3.") || strings.HasPrefix(line, "4.") ||
			strings.HasPrefix(line, "5."); matched {
			// Extract task title after the number
			parts := strings.SplitN(line, ".", 2)
			if len(parts) == 2 {
				taskTitle := strings.TrimSpace(parts[1])
				description := extractTaskDescription(lines, i+1)

				tasks = append(tasks, TaskInfo{
					Title:       taskTitle,
					Description: description,
					Priority:    "medium",
					AgentType:   inferAgentType(taskTitle, filePath),
				})
			}
		}

		// Look for header-based tasks (## Task: Title)
		if strings.HasPrefix(line, "## Task:") || strings.HasPrefix(line, "### Task:") {
			taskTitle := strings.TrimSpace(line[8:]) // Remove "## Task:" prefix
			description := extractTaskDescription(lines, i+1)

			tasks = append(tasks, TaskInfo{
				Title:       taskTitle,
				Description: description,
				Priority:    "high",
				AgentType:   inferAgentType(taskTitle, filePath),
			})
		}
	}

	// If no explicit tasks found, create a general implementation task based on the file
	if len(tasks) == 0 && filePath != "README.md" {
		title := fmt.Sprintf("Implement %s", strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath)))
		tasks = append(tasks, TaskInfo{
			Title:       title,
			Description: fmt.Sprintf("Implement the functionality described in %s", filePath),
			Priority:    "medium",
			AgentType:   inferAgentType(title, filePath),
		})
	}

	return tasks
}

// extractTaskDescription extracts description text following a task
func extractTaskDescription(lines []string, startIndex int) string {
	var description []string

	for i := startIndex; i < len(lines) && i < startIndex+3; i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			break
		}
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") ||
			strings.HasPrefix(line, "#") {
			break
		}
		description = append(description, line)
	}

	return strings.Join(description, " ")
}

// inferAgentType infers the appropriate agent type based on task content
func inferAgentType(taskTitle, filePath string) string {
	taskLower := strings.ToLower(taskTitle)
	pathLower := strings.ToLower(filePath)

	// Backend/API tasks
	if strings.Contains(taskLower, "api") || strings.Contains(taskLower, "server") ||
		strings.Contains(taskLower, "database") || strings.Contains(pathLower, "backend") {
		return "backend-specialist"
	}

	// Frontend tasks
	if strings.Contains(taskLower, "ui") || strings.Contains(taskLower, "frontend") ||
		strings.Contains(taskLower, "component") || strings.Contains(pathLower, "frontend") {
		return "frontend-specialist"
	}

	// DevOps/Infrastructure tasks
	if strings.Contains(taskLower, "deploy") || strings.Contains(taskLower, "docker") ||
		strings.Contains(taskLower, "infrastructure") || strings.Contains(taskLower, "ci/cd") {
		return "devops-specialist"
	}

	// Testing tasks
	if strings.Contains(taskLower, "test") || strings.Contains(taskLower, "qa") {
		return "qa-specialist"
	}

	// Documentation tasks
	if strings.Contains(taskLower, "document") || strings.Contains(pathLower, "readme") ||
		strings.Contains(pathLower, "docs") {
		return "documentation-specialist"
	}

	// Default to general worker
	return "worker"
}

// interactiveReview allows user to review and edit refined files
func interactiveReview(files []RefinedFile) error {
	fmt.Printf("\n📋 Interactive Review Mode\n")
	fmt.Printf("Press Enter to continue, 's' to skip a file, 'q' to quit\n\n")

	for i, file := range files {
		fmt.Printf("File %d/%d: %s\n", i+1, len(files), file.Path)
		fmt.Printf("Preview (first 10 lines):\n")

		lines := strings.Split(file.Content, "\n")
		for j := 0; j < 10 && j < len(lines); j++ {
			fmt.Printf("  %s\n", lines[j])
		}

		fmt.Print("\nAction [Enter/s/q]: ")

		var input string
		fmt.Scanln(&input)

		switch strings.ToLower(input) {
		case "q":
			return gerror.New(gerror.ErrCodeCancelled, "review cancelled by user", nil).
				WithComponent("cli").
				WithOperation("commission.refine.interactiveReview")
		case "s":
			continue
		default:
			// Continue to next file
		}
	}

	return nil
}
