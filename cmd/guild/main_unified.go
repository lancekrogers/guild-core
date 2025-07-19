// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

//go:build unified
// +build unified

// This file demonstrates the unified CLI entry point using the bootstrap system
// To use this version, build with: go build -tags unified

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/lancekrogers/guild/internal/integration/bootstrap"
	"github.com/lancekrogers/guild/internal/integration/services"
	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/observability"
)

// Service mode flags
var (
	enabledServices []string
	configFile      string
)

// unifiedRootCmd is the root command using unified bootstrap
var unifiedRootCmd = &cobra.Command{
	Use:   "guild",
	Short: "Guild - Unified Agent Orchestration Framework",
	Long: `Guild coordinates specialized agents using a unified service architecture.

This unified version uses the bootstrap system for all modes of operation:
- chat: Interactive chat with agents (enables UI services)
- serve: Run the daemon server (enables all services)
- commission: Execute agent commissions (enables agent services)

All commands use the same initialization, configuration, and lifecycle management.`,
}

// unifiedChatCmd runs chat mode with specific services
var unifiedChatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start interactive chat session (unified mode)",
	Long:  `Start an interactive chat session using the unified service architecture.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWithServices(cmd.Context(), []string{
			"memory-service",
			"session-service",
			"kanban-service",
			"chat-ui-service",
			"ui-event-bridge",
		})
	},
}

// unifiedServeCmd runs daemon mode with all services
var unifiedServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the Guild daemon server (unified mode)",
	Long:  `Start the Guild daemon server with all services enabled.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWithServices(cmd.Context(), []string{
			"*", // Enable all services
		})
	},
}

// unifiedCommissionCmd runs commission mode with agent services
var unifiedCommissionCmd = &cobra.Command{
	Use:   "commission [description]",
	Short: "Execute an agent commission (unified mode)",
	Long:  `Execute an agent commission using the unified service architecture.`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Store commission description for later use
		commissionDesc := args[0]
		cmd.SetContext(context.WithValue(cmd.Context(), "commission", commissionDesc))

		return runWithServices(cmd.Context(), []string{
			"memory-service",
			"kanban-service",
			"orchestrator-service",
			"agent-manager-service",
		})
	},
}

// unifiedStatusCmd shows status of all services
var unifiedStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of Guild services",
	Long:  `Display the health and status of all Guild services.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Quick status check without full bootstrap
		fmt.Println("Guild Service Status (Unified Mode)")
		fmt.Println("===================================")

		// TODO: Connect to running daemon and query service status
		fmt.Println("Status checking not yet implemented in unified mode.")
		fmt.Println("This will query the service registry for health status.")

		return nil
	},
}

func init() {
	// Global flags
	unifiedRootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "Config file (default: .guild/guild.yaml)")
	unifiedRootCmd.PersistentFlags().StringSliceVar(&enabledServices, "services", []string{}, "Specific services to enable")

	// Add commands
	unifiedRootCmd.AddCommand(unifiedChatCmd)
	unifiedRootCmd.AddCommand(unifiedServeCmd)
	unifiedRootCmd.AddCommand(unifiedCommissionCmd)
	unifiedRootCmd.AddCommand(unifiedStatusCmd)
}

// runWithServices runs the application with specified services
func runWithServices(ctx context.Context, serviceNames []string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	logger := observability.GetLogger(ctx)
	logger.InfoContext(ctx, "Starting Guild in unified mode",
		"services", serviceNames,
		"config", configFile)

	// Create application with bootstrap
	opts := bootstrap.DefaultOptions()
	if configFile != "" {
		opts.ConfigPath = configFile
	}

	app, err := bootstrap.NewApplication(opts)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create application").
			WithComponent("unified-cli")
	}

	// Initialize application
	if err := app.Initialize(ctx); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize application").
			WithComponent("unified-cli")
	}

	// Configure services based on mode
	if err := configureServices(app, serviceNames); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to configure services").
			WithComponent("unified-cli")
	}

	// Start application
	if err := app.Start(ctx); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to start application").
			WithComponent("unified-cli")
	}

	// Handle mode-specific logic
	if err := handleModeSpecific(ctx, app); err != nil {
		return err
	}

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for interrupt
	select {
	case sig := <-sigChan:
		logger.InfoContext(ctx, "Received signal", "signal", sig)
	case <-ctx.Done():
		logger.InfoContext(ctx, "Context cancelled")
	}

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), opts.ShutdownTimeout)
	defer cancel()

	if err := app.Stop(shutdownCtx); err != nil {
		logger.ErrorContext(shutdownCtx, "Error during shutdown", "error", err)
		return err
	}

	logger.InfoContext(ctx, "Guild shutdown complete")
	return nil
}

// configureServices configures which services to enable
func configureServices(app *bootstrap.Application, serviceNames []string) error {
	// If "*" is specified, all services are enabled by default
	if len(serviceNames) == 1 && serviceNames[0] == "*" {
		return nil // All services registered by default
	}

	// Otherwise, we need to selectively enable services
	// This would involve:
	// 1. Disabling auto-registration in bootstrap
	// 2. Manually registering only requested services
	// 3. Setting up dependencies correctly

	// For now, this is a placeholder
	// TODO: Implement selective service registration

	return nil
}

// handleModeSpecific handles mode-specific logic after services are started
func handleModeSpecific(ctx context.Context, app *bootstrap.Application) error {
	// Check if we're in chat mode
	for _, svc := range app.ServiceRegistry.ListServices() {
		if svc == "chat-ui-service" {
			// Get the chat UI service and run it
			// This would block until the UI exits
			// TODO: Get service from registry and call Run()
			logger := observability.GetLogger(ctx)
			logger.InfoContext(ctx, "Chat UI service detected, starting interactive mode")

			// Placeholder for actual implementation
			fmt.Println("Chat UI would start here in unified mode")
			fmt.Println("Press Ctrl+C to exit")

			return nil
		}
	}

	// Check if we have a commission to execute
	if commission, ok := ctx.Value("commission").(string); ok {
		logger := observability.GetLogger(ctx)
		logger.InfoContext(ctx, "Executing commission", "description", commission)

		// TODO: Get orchestrator service and submit commission
		fmt.Printf("Would execute commission: %s\n", commission)

		// Commission mode exits after submission
		return nil
	}

	// Daemon mode - just keep running
	return nil
}

// main function for unified mode
func main() {
	ctx := context.Background()

	// Set up observability
	logger := observability.NewLogger(nil)
	ctx = observability.WithLogger(ctx, logger)
	ctx = observability.EnsureRequestContext(ctx)
	ctx = observability.WithComponent(ctx, "guild-unified")

	// Execute root command
	unifiedRootCmd.SetContext(ctx)
	if err := unifiedRootCmd.Execute(); err != nil {
		logger.ErrorContext(ctx, "Command failed", "error", err)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
