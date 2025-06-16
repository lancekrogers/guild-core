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
	"syscall"

	"github.com/spf13/cobra"

	"github.com/guild-ventures/guild-core/internal/daemon"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	grpcpkg "github.com/guild-ventures/guild-core/pkg/grpc"
	"github.com/guild-ventures/guild-core/pkg/project"
	"github.com/guild-ventures/guild-core/pkg/registry"
)

var (
	servePort   string
	serveHost   string
	serveDaemon bool
)

func init() {
	serveCmd.Flags().StringVar(&servePort, "port", "9090", "Port to serve gRPC on")
	serveCmd.Flags().StringVar(&serveHost, "host", "localhost", "Host to serve gRPC on")
	serveCmd.Flags().BoolVar(&serveDaemon, "daemon", false, "Run as background daemon")
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

The server must be running for the 'guild chat' command to work.

Usage:
  Terminal 1: guild serve
  Terminal 2: guild chat`,
	RunE: runServe,
}

func runServe(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// If running as daemon, set up logging to file
	if serveDaemon {
		logPath := daemon.GetLogFilePath()
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
	_, err := project.GetContext()
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
	serverAddr := fmt.Sprintf("%s:%s", serveHost, servePort)

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
		log.Printf("Guild gRPC server (daemon) starting at %s\n", serverAddr)
		log.Println("Running in daemon mode - output redirected to:", daemon.GetLogFilePath())
	} else {
		fmt.Printf("🏰 Guild gRPC server running at %s\n", serverAddr)
		fmt.Println("📡 Registered gRPC services:")
		fmt.Println("   ✅ Guild Service (campaigns, agents, commissions)")
		fmt.Println("   ✅ Chat Service (interactive agent communication)")
		fmt.Println("   ✅ Prompt Service (prompt management)")
		fmt.Println()
		fmt.Println("💡 Use 'guild chat' in another terminal to connect")
		fmt.Println("🛑 Press Ctrl+C to stop the server")
	}

	// Start the server (this blocks until context is cancelled)
	if err := server.Start(ctx, serverAddr); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeConnection, "failed to start gRPC server").
			WithComponent("cli").
			WithOperation("serve.run").
			WithDetails("server_address", serverAddr)
	}

	if !serveDaemon {
		fmt.Println("✨ Guild server stopped gracefully...done.")
	} else {
		log.Println("Guild server stopped gracefully")
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
