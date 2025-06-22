// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/guild-ventures/guild-core/internal/daemon"
	"github.com/guild-ventures/guild-core/pkg/campaign"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	grpcpkg "github.com/guild-ventures/guild-core/pkg/grpc"
	"github.com/guild-ventures/guild-core/pkg/project"
	"github.com/guild-ventures/guild-core/pkg/registry"
)

var (
	serveDaemon   bool
	serveCampaign string
	serveSession  string
	serveSocket   string
)

func init() {
	serveCmd.Flags().BoolVar(&serveDaemon, "daemon", false, "Run as background daemon")
	serveCmd.Flags().StringVar(&serveCampaign, "campaign", "", "Campaign to serve (uses detection if not specified)")
	serveCmd.Flags().StringVar(&serveSession, "session", "0", "Session number for multi-session campaigns")
	serveCmd.Flags().StringVar(&serveSocket, "socket", "", "Unix socket path (overrides auto-generated path)")

	// Register completion functions
	serveCmd.RegisterFlagCompletionFunc("campaign", completeCampaignNames)
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the Guild gRPC server",
	Long: `Start the Guild gRPC server to enable remote communication with agents.

This starts a gRPC server that provides:
- Chat service for interactive agent communication
- Campaign management and monitoring
- Agent status and control
- Prompt management services

Campaign Architecture:
- Use --campaign to specify which campaign to serve
- Without --campaign, detects campaign from current directory
- Each campaign runs on its own Unix socket
- Supports multiple concurrent sessions per campaign

Examples:
  guild serve                           # Auto-detect campaign, start server
  guild serve --campaign e-commerce     # Serve specific campaign
  guild serve --daemon                  # Run as background daemon
  guild serve --session 1               # Start additional session`,
	RunE: runServe,
}

func runServe(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Detect campaign if not explicitly provided
	cwd, err := os.Getwd()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get current directory").
			WithComponent("cli").
			WithOperation("serve.run")
	}

	campaignName, err := campaign.DetectCampaign(cwd, serveCampaign)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInvalidInput, "failed to detect campaign").
			WithComponent("cli").
			WithOperation("serve.run").
			WithDetails("help", "Make sure you're in a campaign directory or specify --campaign")
	}

	// Parse session number
	sessionNum := 0
	if serveSession != "" && serveSession != "0" {
		if parsed, err := strconv.Atoi(serveSession); err == nil {
			sessionNum = parsed
		} else {
			return gerror.Wrap(err, gerror.ErrCodeInvalidInput, "invalid session number").
				WithComponent("cli").
				WithOperation("serve.run")
		}
	}

	// Get daemon configuration
	daemonConfig, err := daemon.GetDaemonConfig(campaignName, sessionNum)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get daemon config").
			WithComponent("cli").
			WithOperation("serve.run")
	}

	// Override socket path if provided
	if serveSocket != "" {
		daemonConfig.SocketPath = serveSocket
	}

	// If running as daemon, set up logging to file
	if serveDaemon {
		logPath := daemonConfig.LogFile
		if logPath == "" {
			logPath = daemon.GetLogFilePath()
		}
		logDir := filepath.Dir(logPath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeIO, "failed to create log directory").
				WithComponent("cli").
				WithOperation("serve.daemon")
		}

		logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeIO, "failed to open log file").
				WithComponent("cli").
				WithOperation("serve.daemon")
		}
		defer logFile.Close()

		// Redirect stdout and stderr to log file
		log.SetOutput(logFile)
		os.Stdout = logFile
		os.Stderr = logFile
	}

	// Initialize project
	_, err = project.GetContext()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to load project").
			WithComponent("cli").
			WithOperation("serve.run")
	}

	// Initialize registry with minimal working configuration
	reg := registry.NewComponentRegistry()
	registryConfig := &registry.Config{
		Agents: registry.AgentConfigYaml{
			DefaultType: "worker",
			Types: map[string]interface{}{
				"worker": map[string]interface{}{
					"enabled": true,
				},
			},
		},
		Tools: registry.ToolConfig{
			EnabledTools: []string{"file", "shell", "http"},
			Settings: map[string]interface{}{
				"timeout": "30s",
			},
		},
		Providers: registry.ProviderConfig{
			DefaultProvider: "claudecode",
			Providers: map[string]interface{}{
				"claudecode": map[string]interface{}{
					"model":    "sonnet",
					"bin_path": "claude-code",
				},
			},
		},
		Memory: registry.MemoryConfig{
			DefaultMemoryStore: "sqlite",
			DefaultVectorStore: "chromem",
			Stores: map[string]interface{}{
				"sqlite": map[string]interface{}{
					"path": "./.guild/memory.db",
				},
				"chromem": map[string]interface{}{
					"persistence_path": "./.guild/vectors",
					"dimension":        1536,
				},
			},
		},
	}

	if err := reg.Initialize(context.Background(), *registryConfig); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize registry").
			WithComponent("cli").
			WithOperation("serve.run")
	}

	// Create event bus (memory-based for now)
	eventBus := &memoryEventBus{}

	// Create gRPC server
	server := grpcpkg.NewServer(reg, eventBus)
	serverAddr := daemonConfig.GetServerAddress()

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		if !serveDaemon {
			fmt.Println("\n🛑 Shutting down gRPC server...")
		}
		cancel()
	}()

	// Log startup information
	if serveDaemon {
		log.Printf("%s starting at %s\n", daemonConfig.GetDisplayName(), serverAddr)
		log.Printf("Campaign: %s, Session: %d\n", campaignName, sessionNum)
		log.Println("Running in daemon mode - output redirected to:", daemonConfig.LogFile)
	} else {
		fmt.Printf("🏰 %s running at %s\n", daemonConfig.GetDisplayName(), serverAddr)
		fmt.Printf("📋 Campaign: %s (session %d)\n", campaignName, sessionNum)
		fmt.Println("📡 Registered gRPC services:")
		fmt.Println("   ✅ Guild Service (campaigns, agents, commissions)")
		fmt.Println("   ✅ Chat Service (interactive agent communication)")
		fmt.Println("   ✅ Prompt Service (prompt management)")
		fmt.Println()
		fmt.Printf("🔌 Socket: %s\n", daemonConfig.SocketPath)
		fmt.Println("💡 Use 'guild chat --campaign", campaignName, "' to connect")
		fmt.Println("🛑 Press Ctrl+C to stop the server")
	}

	// Start the server (this blocks until context is cancelled)
	// Ensure socket directory exists and clean stale sockets
	socketDir := filepath.Dir(daemonConfig.SocketPath)
	if err := os.MkdirAll(socketDir, 0755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to create socket directory").
			WithComponent("cli").
			WithOperation("serve.run")
	}

	// Remove stale socket file if it exists
	if _, err := os.Stat(daemonConfig.SocketPath); err == nil {
		os.Remove(daemonConfig.SocketPath)
	}

	startErr := server.StartUnix(ctx, daemonConfig.SocketPath)

	if startErr != nil {
		return gerror.Wrap(startErr, gerror.ErrCodeConnection, "failed to start gRPC server").
			WithComponent("cli").
			WithOperation("serve.run").
			WithDetails("server_address", serverAddr)
	}

	if !serveDaemon {
		fmt.Printf("✨ %s stopped gracefully...done.\n", daemonConfig.GetDisplayName())
	} else {
		log.Printf("%s stopped gracefully\n", daemonConfig.GetDisplayName())
	}
	return nil
}

// memoryEventBus is a simple in-memory event bus implementation that matches grpc.EventBus interface
type memoryEventBus struct{}

func (m *memoryEventBus) Publish(event interface{}) {
	// Simple no-op implementation for now
	// In a real implementation, this would broadcast events to subscribers
}

func (m *memoryEventBus) Subscribe(eventType string, handler func(event interface{})) {
	// Simple no-op implementation for now
}
