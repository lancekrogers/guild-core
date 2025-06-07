package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	promptspb "github.com/guild-ventures/guild-core/pkg/grpc/pb/prompts/v1"
)

var (
	promptGrpcAddress = "localhost:9090" // Default gRPC server address
	promptArtisanID   string
	promptSessionID   string
	promptLayer       string
	promptContent     string
	promptOutputJSON  bool
)

// promptCmd represents the prompt command for managing Guild layered prompts
var promptCmd = &cobra.Command{
	Use:   "prompt",
	Short: "Manage Guild layered prompts",
	Long: `Manage Guild layered prompts through the CLI.

Guild uses a layered prompt system with six hierarchical layers:
- platform: Core Guild platform rules (global)
- guild: Project-wide goals and style guidelines
- role: Artisan role definitions (backend, frontend, etc.)
- domain: Project type specializations (web-app, cli-tool, etc.)
- session: User preferences and session-specific context
- turn: Ephemeral instructions for single interactions

Use the subcommands to get, set, list, and delete prompts at different layers.`,
}

// promptGetCmd gets a specific prompt layer
var promptGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a specific prompt layer",
	Long: `Get a specific prompt layer for an artisan and session.

Examples:
  guild prompt get --layer=platform
  guild prompt get --layer=role --artisan-id=backend-dev-001
  guild prompt get --layer=session --session-id=session_123`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPromptGet()
	},
}

// promptSetCmd sets or updates a specific prompt layer
var promptSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set or update a specific prompt layer",
	Long: `Set or update a specific prompt layer for an artisan and session.

Examples:
  guild prompt set --layer=session --session-id=session_123 --content="User prefers detailed explanations"
  guild prompt set --layer=role --artisan-id=backend-dev-001 --content="You are a backend artisan..."`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPromptSet()
	},
}

// promptListCmd lists all prompt layers
var promptListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all prompt layers for an artisan/session",
	Long: `List all prompt layers for an artisan and session.

Examples:
  guild prompt list --artisan-id=backend-dev-001 --session-id=session_123
  guild prompt list --artisan-id=backend-dev-001`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPromptList()
	},
}

// promptDeleteCmd deletes a specific prompt layer
var promptDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a specific prompt layer",
	Long: `Delete a specific prompt layer for an artisan and session.

Examples:
  guild prompt delete --layer=session --session-id=session_123
  guild prompt delete --layer=role --artisan-id=backend-dev-001`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPromptDelete()
	},
}

// promptBuildCmd builds a complete layered prompt
var promptBuildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build a complete layered prompt",
	Long: `Build a complete layered prompt by assembling all relevant layers.

Examples:
  guild prompt build --artisan-id=backend-dev-001 --session-id=session_123
  guild prompt build --artisan-id=frontend-designer-002`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPromptBuild()
	},
}

// promptCacheCmd manages prompt cache
var promptCacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage prompt cache",
	Long: `Manage the Guild layered prompt cache.

Subcommands:
  clear - Clear the prompt cache for an artisan/session`,
}

// promptCacheClearCmd clears the prompt cache
var promptCacheClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear the prompt cache",
	Long: `Clear the prompt cache for an artisan and session.

Examples:
  guild prompt cache clear --artisan-id=backend-dev-001 --session-id=session_123
  guild prompt cache clear --artisan-id=backend-dev-001`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPromptCacheClear()
	},
}

// promptStatsCmd shows statistics for prompt layers
var promptStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show statistics for prompt layers",
	Long: `Show statistics for specific prompt layers.

Examples:
  guild prompt stats --layer=platform
  guild prompt stats --layer=role`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPromptStats()
	},
}

func init() {
	// Add prompt command to root
	rootCmd.AddCommand(promptCmd)

	// Add subcommands
	promptCmd.AddCommand(promptGetCmd)
	promptCmd.AddCommand(promptSetCmd)
	promptCmd.AddCommand(promptListCmd)
	promptCmd.AddCommand(promptDeleteCmd)
	promptCmd.AddCommand(promptBuildCmd)
	promptCmd.AddCommand(promptCacheCmd)
	promptCmd.AddCommand(promptStatsCmd)

	// Add cache subcommands
	promptCacheCmd.AddCommand(promptCacheClearCmd)

	// Global flags
	promptCmd.PersistentFlags().StringVar(&promptGrpcAddress, "grpc-address", "localhost:9090", "gRPC server address")
	promptCmd.PersistentFlags().StringVar(&promptArtisanID, "artisan-id", "", "Artisan ID (e.g., backend-dev-001)")
	promptCmd.PersistentFlags().StringVar(&promptSessionID, "session-id", "", "Session ID")
	promptCmd.PersistentFlags().BoolVar(&promptOutputJSON, "json", false, "Output in JSON format")

	// Command-specific flags
	promptGetCmd.Flags().StringVar(&promptLayer, "layer", "", "Prompt layer (platform, guild, role, domain, session, turn)")
	promptGetCmd.MarkFlagRequired("layer")

	promptSetCmd.Flags().StringVar(&promptLayer, "layer", "", "Prompt layer (platform, guild, role, domain, session, turn)")
	promptSetCmd.Flags().StringVar(&promptContent, "content", "", "Prompt content")
	promptSetCmd.MarkFlagRequired("layer")
	promptSetCmd.MarkFlagRequired("content")

	promptDeleteCmd.Flags().StringVar(&promptLayer, "layer", "", "Prompt layer (platform, guild, role, domain, session, turn)")
	promptDeleteCmd.MarkFlagRequired("layer")

	promptStatsCmd.Flags().StringVar(&promptLayer, "layer", "", "Prompt layer (platform, guild, role, domain, session, turn)")
	promptStatsCmd.MarkFlagRequired("layer")
}

// Helper functions

func getGRPCClient() (promptspb.PromptServiceClient, *grpc.ClientConn, error) {
	conn, err := grpc.Dial(promptGrpcAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to connect to gRPC server").
			WithComponent("prompt").
			WithOperation("getGRPCClient")
	}

	client := promptspb.NewPromptServiceClient(conn)
	return client, conn, nil
}

func parseLayer(layerStr string) promptspb.PromptLayer {
	switch strings.ToLower(layerStr) {
	case "platform":
		return promptspb.PromptLayer_PROMPT_LAYER_PLATFORM
	case "guild":
		return promptspb.PromptLayer_PROMPT_LAYER_GUILD
	case "role":
		return promptspb.PromptLayer_PROMPT_LAYER_ROLE
	case "domain":
		return promptspb.PromptLayer_PROMPT_LAYER_DOMAIN
	case "session":
		return promptspb.PromptLayer_PROMPT_LAYER_SESSION
	case "turn":
		return promptspb.PromptLayer_PROMPT_LAYER_TURN
	default:
		return promptspb.PromptLayer_PROMPT_LAYER_UNSPECIFIED
	}
}

func layerToString(layer promptspb.PromptLayer) string {
	switch layer {
	case promptspb.PromptLayer_PROMPT_LAYER_PLATFORM:
		return "platform"
	case promptspb.PromptLayer_PROMPT_LAYER_GUILD:
		return "guild"
	case promptspb.PromptLayer_PROMPT_LAYER_ROLE:
		return "role"
	case promptspb.PromptLayer_PROMPT_LAYER_DOMAIN:
		return "domain"
	case promptspb.PromptLayer_PROMPT_LAYER_SESSION:
		return "session"
	case promptspb.PromptLayer_PROMPT_LAYER_TURN:
		return "turn"
	default:
		return "unknown"
	}
}

// Command implementations

func runPromptGet() error {
	client, conn, err := getGRPCClient()
	if err != nil {
		return err
	}
	defer conn.Close()

	layerEnum := parseLayer(promptLayer)
	if layerEnum == promptspb.PromptLayer_PROMPT_LAYER_UNSPECIFIED {
		return gerror.New(gerror.ErrCodeMissingRequired, fmt.Sprintf("invalid layer: %s", promptLayer), nil).
			WithComponent("prompt").WithOperation("runPromptGet")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.GetPromptLayer(ctx, &promptspb.GetPromptLayerRequest{
		Layer:     layerEnum,
		ArtisanId: promptArtisanID,
		SessionId: promptSessionID,
	})
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get prompt layer").WithComponent("prompt").WithOperation("runPromptGet")
	}

	if promptOutputJSON {
		data, err := json.MarshalIndent(resp.Prompt, "", "  ")
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal JSON").
				WithComponent("cli").
				WithOperation("jsonOutput")
		}
		fmt.Println(string(data))
	} else {
		fmt.Printf("Layer: %s\n", layerToString(resp.Prompt.Layer))
		fmt.Printf("Artisan ID: %s\n", resp.Prompt.ArtisanId)
		fmt.Printf("Session ID: %s\n", resp.Prompt.SessionId)
		fmt.Printf("Version: %d\n", resp.Prompt.Version)
		fmt.Printf("Priority: %d\n", resp.Prompt.Priority)
		fmt.Printf("Updated: %s\n", resp.Prompt.Updated.AsTime().Format(time.RFC3339))
		fmt.Printf("\nContent:\n%s\n", resp.Prompt.Content)
		if len(resp.Prompt.Metadata) > 0 {
			fmt.Printf("\nMetadata:\n")
			for k, v := range resp.Prompt.Metadata {
				fmt.Printf("  %s: %s\n", k, v)
			}
		}
	}

	return nil
}

func runPromptSet() error {
	client, conn, err := getGRPCClient()
	if err != nil {
		return err
	}
	defer conn.Close()

	layerEnum := parseLayer(promptLayer)
	if layerEnum == promptspb.PromptLayer_PROMPT_LAYER_UNSPECIFIED {
		return gerror.New(gerror.ErrCodeMissingRequired, fmt.Sprintf("invalid layer: %s", promptLayer), nil).
			WithComponent("prompt").WithOperation("runPromptSet")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.SetPromptLayer(ctx, &promptspb.SetPromptLayerRequest{
		Prompt: &promptspb.SystemPrompt{
			Layer:     layerEnum,
			ArtisanId: promptArtisanID,
			SessionId: promptSessionID,
			Content:   promptContent,
			Version:   1,
		},
	})
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to set prompt layer").WithComponent("prompt").WithOperation("runPromptSet")
	}

	if promptOutputJSON {
		data, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal JSON").WithComponent("cli").WithOperation("jsonOutput")
		}
		fmt.Println(string(data))
	} else {
		fmt.Printf("✅ %s\n", resp.Message)
	}

	return nil
}

func runPromptList() error {
	client, conn, err := getGRPCClient()
	if err != nil {
		return err
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.ListPromptLayers(ctx, &promptspb.ListPromptLayersRequest{
		ArtisanId: promptArtisanID,
		SessionId: promptSessionID,
	})
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to list prompt layers").WithComponent("prompt").WithOperation("runPromptList")
	}

	if promptOutputJSON {
		data, err := json.MarshalIndent(resp.Prompts, "", "  ")
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal JSON").WithComponent("cli").WithOperation("jsonOutput")
		}
		fmt.Println(string(data))
	} else {
		if len(resp.Prompts) == 0 {
			fmt.Println("No prompt layers found.")
			return nil
		}

		fmt.Printf("Found %d prompt layers:\n\n", len(resp.Prompts))
		for i, prompt := range resp.Prompts {
			fmt.Printf("%d. Layer: %s\n", i+1, layerToString(prompt.Layer))
			if prompt.ArtisanId != "" {
				fmt.Printf("   Artisan: %s\n", prompt.ArtisanId)
			}
			if prompt.SessionId != "" {
				fmt.Printf("   Session: %s\n", prompt.SessionId)
			}
			fmt.Printf("   Version: %d, Priority: %d\n", prompt.Version, prompt.Priority)
			fmt.Printf("   Updated: %s\n", prompt.Updated.AsTime().Format(time.RFC3339))

			// Show content preview
			contentPreview := prompt.Content
			if len(contentPreview) > 100 {
				contentPreview = contentPreview[:100] + "..."
			}
			fmt.Printf("   Content: %s\n", strings.ReplaceAll(contentPreview, "\n", "\\n"))
			fmt.Println()
		}
	}

	return nil
}

func runPromptDelete() error {
	client, conn, err := getGRPCClient()
	if err != nil {
		return err
	}
	defer conn.Close()

	layerEnum := parseLayer(promptLayer)
	if layerEnum == promptspb.PromptLayer_PROMPT_LAYER_UNSPECIFIED {
		return gerror.New(gerror.ErrCodeMissingRequired, fmt.Sprintf("invalid layer: %s", promptLayer), nil).
			WithComponent("prompt").WithOperation("runPromptDelete")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.DeletePromptLayer(ctx, &promptspb.DeletePromptLayerRequest{
		Layer:     layerEnum,
		ArtisanId: promptArtisanID,
		SessionId: promptSessionID,
	})
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to delete prompt layer").WithComponent("prompt").WithOperation("runPromptDelete")
	}

	if promptOutputJSON {
		data, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal JSON").WithComponent("cli").WithOperation("jsonOutput")
		}
		fmt.Println(string(data))
	} else {
		fmt.Printf("✅ %s\n", resp.Message)
	}

	return nil
}

func runPromptBuild() error {
	client, conn, err := getGRPCClient()
	if err != nil {
		return err
	}
	defer conn.Close()

	if promptArtisanID == "" {
		return gerror.New(gerror.ErrCodeMissingRequired, "artisan-id is required for building layered prompts", nil).WithComponent("prompt").WithOperation("runPromptBuild")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.BuildLayeredPrompt(ctx, &promptspb.BuildLayeredPromptRequest{
		ArtisanId: promptArtisanID,
		SessionId: promptSessionID,
	})
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to build layered prompt").WithComponent("prompt").WithOperation("runPromptBuild")
	}

	if promptOutputJSON {
		data, err := json.MarshalIndent(resp.Prompt, "", "  ")
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal JSON").WithComponent("cli").WithOperation("jsonOutput")
		}
		fmt.Println(string(data))
	} else {
		prompt := resp.Prompt
		fmt.Printf("Layered Prompt for %s\n", prompt.ArtisanId)
		if prompt.SessionId != "" {
			fmt.Printf("Session: %s\n", prompt.SessionId)
		}
		fmt.Printf("Layers: %d, Tokens: %d", len(prompt.Layers), prompt.TokenCount)
		if prompt.Truncated {
			fmt.Printf(" (truncated)")
		}
		fmt.Printf("\nAssembled: %s\n", prompt.AssembledAt.AsTime().Format(time.RFC3339))
		fmt.Printf("Cache Key: %s\n\n", prompt.CacheKey)

		fmt.Println("=== COMPILED PROMPT ===")
		fmt.Println(prompt.Compiled)
		fmt.Println("========================")

		if len(prompt.Layers) > 0 {
			fmt.Printf("\n=== LAYER BREAKDOWN ===\n")
			for i, layer := range prompt.Layers {
				fmt.Printf("%d. %s (Priority: %d)\n", i+1, layerToString(layer.Layer), layer.Priority)
			}
		}
	}

	return nil
}

func runPromptCacheClear() error {
	client, conn, err := getGRPCClient()
	if err != nil {
		return err
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.InvalidateCache(ctx, &promptspb.InvalidateCacheRequest{
		ArtisanId: promptArtisanID,
		SessionId: promptSessionID,
	})
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to clear cache").WithComponent("prompt").WithOperation("runPromptClearCache")
	}

	if promptOutputJSON {
		data, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal JSON").WithComponent("cli").WithOperation("jsonOutput")
		}
		fmt.Println(string(data))
	} else {
		fmt.Printf("✅ %s\n", resp.Message)
	}

	return nil
}

func runPromptStats() error {
	client, conn, err := getGRPCClient()
	if err != nil {
		return err
	}
	defer conn.Close()

	layerEnum := parseLayer(promptLayer)
	if layerEnum == promptspb.PromptLayer_PROMPT_LAYER_UNSPECIFIED {
		return gerror.New(gerror.ErrCodeMissingRequired, fmt.Sprintf("invalid layer: %s", promptLayer), nil).
			WithComponent("prompt").WithOperation("runPromptStats")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.GetLayerStats(ctx, &promptspb.GetLayerStatsRequest{
		Layer: layerEnum,
	})
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get layer stats").WithComponent("prompt").WithOperation("runPromptStats")
	}

	if promptOutputJSON {
		data, err := json.MarshalIndent(resp.Stats, "", "  ")
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal JSON").WithComponent("cli").WithOperation("jsonOutput")
		}
		fmt.Println(string(data))
	} else {
		stats := resp.Stats
		fmt.Printf("Statistics for %s layer:\n", layerToString(stats.Layer))
		fmt.Printf("  Prompt Count: %d\n", stats.PromptCount)
		fmt.Printf("  Average Tokens: %d\n", stats.AverageTokens)
		fmt.Printf("  Last Updated: %s\n", stats.LastUpdated.AsTime().Format(time.RFC3339))
	}

	return nil
}
