package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/prompts"
)

// GuildMasterRefiner implements the CommissionRefiner interface
type GuildMasterRefiner struct {
	artisanClient ArtisanClient
	promptManager prompts.Manager
	parser        ResponseParser
	validator     StructureValidator
}

// NewGuildMasterRefiner creates a new Guild Master refiner
func NewGuildMasterRefiner(
	artisanClient ArtisanClient,
	promptManager prompts.Manager,
	parser ResponseParser,
	validator StructureValidator,
) *GuildMasterRefiner {
	return &GuildMasterRefiner{
		artisanClient: artisanClient,
		promptManager: promptManager,
		parser:        parser,
		validator:     validator,
	}
}

// RefineCommission implements the CommissionRefiner interface
func (r *GuildMasterRefiner) RefineCommission(ctx context.Context, commission Commission) (*RefinedCommission, error) {
	// Get the appropriate system prompt based on domain
	systemPrompt, err := r.getSystemPrompt(ctx, commission.Domain)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get system prompt").
			WithComponent("manager").
			WithOperation("RefineCommission").
			WithDetails("domain", commission.Domain)
	}

	// Prepare the user prompt with commission details
	userPrompt := r.buildUserPrompt(commission)

	// Call the Guild Artisan for refinement
	response, err := r.artisanClient.Complete(ctx, ArtisanRequest{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		Temperature:  0.7,
		MaxTokens:    4000,
	})
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeAgent, "failed to get Artisan response").
			WithComponent("manager").
			WithOperation("RefineCommission").
			WithDetails("commission_id", commission.ID)
	}

	// Parse the response into a file structure for the Archives
	structure, err := r.parser.ParseResponse(response)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to parse Artisan response").
			WithComponent("manager").
			WithOperation("RefineCommission").
			WithDetails("commission_id", commission.ID)
	}

	// Validate the structure meets Guild standards
	if err := r.validator.ValidateStructure(structure); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "structure does not meet Guild standards").
			WithComponent("manager").
			WithOperation("RefineCommission").
			WithDetails("commission_id", commission.ID)
	}

	// Create the refined commission
	refined := &RefinedCommission{
		CommissionID: commission.ID,
		Structure:    structure,
		Metadata: map[string]interface{}{
			"domain":               commission.Domain,
			"original_title":       commission.Title,
			"refinement_timestamp": ctx.Value("timestamp"),
			"guild_master":         "auto-refiner",
		},
	}

	return refined, nil
}

// RefineCommissionSimple provides a simplified interface for CLI usage
func (r *GuildMasterRefiner) RefineCommissionSimple(ctx context.Context, commissionText string, domain string) (string, error) {
	// Get the appropriate system prompt based on domain
	systemPrompt, err := r.getSystemPrompt(ctx, domain)
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get system prompt").
			WithComponent("manager").
			WithOperation("RefineCommissionSimple").
			WithDetails("domain", domain)
	}

	// Create simple user prompt
	userPrompt := fmt.Sprintf("Please refine the following commission into a hierarchical implementation plan:\n\n%s", commissionText)

	// Call the Guild Artisan for refinement
	response, err := r.artisanClient.Complete(ctx, ArtisanRequest{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		Temperature:  0.7,
		MaxTokens:    4000,
	})
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeAgent, "failed to get Artisan response").
			WithComponent("manager").
			WithOperation("RefineCommissionSimple")
	}

	return response.Content, nil
}

// getSystemPrompt retrieves the appropriate Guild Master system prompt
func (r *GuildMasterRefiner) getSystemPrompt(ctx context.Context, domain string) (string, error) {
	// Default to "default" domain if empty
	if domain == "" {
		domain = "default"
	}

	prompt, err := r.promptManager.GetSystemPrompt(ctx, "manager", domain)
	if err != nil {
		return "", gerror.Wrapf(err, gerror.ErrCodeInternal, "failed to get Guild Master prompt for domain %s", domain).
			WithComponent("manager").
			WithOperation("getSystemPrompt").
			WithDetails("domain", domain)
	}

	return prompt, nil
}

// buildUserPrompt creates the user prompt with commission details
func (r *GuildMasterRefiner) buildUserPrompt(commission Commission) string {
	var builder strings.Builder

	builder.WriteString("Guild Master, please refine the following commission into a detailed implementation plan for our artisans:\n\n")
	builder.WriteString(fmt.Sprintf("**Commission ID**: %s\n", commission.ID))
	builder.WriteString(fmt.Sprintf("**Title**: %s\n", commission.Title))
	builder.WriteString(fmt.Sprintf("**Commission Description**:\n%s\n", commission.Description))

	// Add any additional context
	if len(commission.Context) > 0 {
		builder.WriteString("\n**Additional Guild Context**:\n")
		for key, value := range commission.Context {
			// Try to convert to string, otherwise JSON encode
			if strVal, ok := value.(string); ok {
				builder.WriteString(fmt.Sprintf("- %s: %s\n", key, strVal))
			} else {
				jsonVal, _ := json.Marshal(value)
				builder.WriteString(fmt.Sprintf("- %s: %s\n", key, string(jsonVal)))
			}
		}
	}

	builder.WriteString("\nPlease create a hierarchical directory structure with markdown files that breaks down this commission into implementable tasks for our artisans. Follow the format specified in your Guild Master prompt. Each task should be suitable for assignment to specialized artisans through the Workshop Board.")

	return builder.String()
}

// GetArtisanClient returns the artisan client for external use
func (r *GuildMasterRefiner) GetArtisanClient() ArtisanClient {
	return r.artisanClient
}