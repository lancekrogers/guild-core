// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/guild-framework/guild-core/internal/daemon"
	"github.com/guild-framework/guild-core/pkg/campaign"
	"github.com/guild-framework/guild-core/pkg/events"
	"github.com/guild-framework/guild-core/pkg/gerror"
	grpcpkg "github.com/guild-framework/guild-core/pkg/grpc"
	"github.com/guild-framework/guild-core/pkg/observability"
	"github.com/guild-framework/guild-core/pkg/project"
	"github.com/guild-framework/guild-core/pkg/registry"
)

var (
	serveForeground bool
	serveDev        bool
	serveCampaign   string
	serveSession    string
	serveSocket     string
	serveConfig     string
)

func init() {
	serveCmd.Flags().BoolVar(&serveForeground, "foreground", false, "Run in foreground (default: background)")
	serveCmd.Flags().BoolVar(&serveDev, "dev", false, "Run in development mode (foreground, verbose logging)")
	serveCmd.Flags().StringVar(&serveCampaign, "campaign", "", "Campaign to serve (uses detection if not specified)")
	serveCmd.Flags().StringVar(&serveSession, "session", "0", "Session number for multi-session campaigns")
	serveCmd.Flags().StringVar(&serveSocket, "socket", "", "Unix socket path (overrides auto-generated path)")
	serveCmd.Flags().StringVar(&serveConfig, "config", "", "Path to daemon config file")

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

The server runs in the background by default. Use --foreground to run interactively.

Campaign Architecture:
- Use --campaign to specify which campaign to serve
- Without --campaign, detects campaign from current directory
- Each campaign runs on its own Unix socket
- Supports multiple concurrent sessions per campaign

Examples:
  guild serve                           # Auto-detect campaign, start background server
  guild serve --dev                     # Run in development mode (foreground, verbose)
  guild serve --foreground              # Run in foreground with output
  guild serve --campaign e-commerce     # Serve specific campaign
  guild serve --session 1               # Start additional session
  guild serve --config daemon.yaml      # Use custom config file`,
	RunE: runServe,
}

func runServe(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	// Check context early
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "serve command cancelled").
			WithComponent("cli").
			WithOperation("serve.run")
	}

	// Handle --dev flag (implies foreground and verbose logging)
	if serveDev {
		serveForeground = true
		// TODO: Set verbose logging level when observability supports it
	}

	// Set up logging
	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "guild-serve")
	ctx = observability.WithOperation(ctx, "runServe")

	logger.InfoContext(ctx, "Starting Guild gRPC server",
		"foreground", serveForeground,
		"dev", serveDev,
		"campaign", serveCampaign,
		"session", serveSession,
		"socket", serveSocket,
		"config", serveConfig,
	)

	// Detect campaign if not explicitly provided
	cwd, err := os.Getwd()
	if err != nil {
		logger.ErrorContext(ctx, "Failed to get current directory", "error", err)
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get current directory").
			WithComponent("cli").
			WithOperation("serve.run")
	}

	campaignName, err := campaign.DetectCampaign(cwd, serveCampaign)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to detect campaign", "error", err, "cwd", cwd, "specified_campaign", serveCampaign)
		return gerror.Wrap(err, gerror.ErrCodeInvalidInput, "failed to detect campaign").
			WithComponent("cli").
			WithOperation("serve.run").
			WithDetails("help", "Make sure you're in a campaign directory or specify --campaign")
	}

	logger.InfoContext(ctx, "Detected campaign", "campaign", campaignName, "cwd", cwd)

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

	// Load YAML configuration if specified
	if serveConfig != "" {
		yamlConfig, err := daemon.LoadYAMLConfig(serveConfig)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to load config file").
				WithComponent("cli").
				WithOperation("serve.run").
				WithDetails("config", serveConfig)
		}

		// Apply YAML config to daemon config
		yamlConfig.ApplyToConfig(daemonConfig)

		logger.InfoContext(ctx, "Loaded configuration from file", "config", serveConfig)
	}

	// Override socket path if provided via command line (takes precedence)
	if serveSocket != "" {
		daemonConfig.SocketPath = serveSocket
	}

	// If running in background, set up logging to file
	if !serveForeground {
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

		logger.InfoContext(ctx, "Running as daemon, logs redirected to file", "log_path", logPath)
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
	logger.InfoContext(ctx, "Initializing component registry")
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
					"path": "./.campaign/memory.db",
				},
				"chromem": map[string]interface{}{
					"persistence_path": "./.campaign/vectors",
					"dimension":        1536,
				},
			},
		},
	}

	if err := reg.Initialize(ctx, *registryConfig); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize registry").
			WithComponent("cli").
			WithOperation("serve.run")
	}

	// Create event bus using the unified events package
	unifiedEventBus := events.NewMemoryEventBusWithDefaults()

	// Wrap it with adapter for grpc compatibility
	eventBusAdapter := grpcpkg.NewEventBusAdapter(unifiedEventBus)

	// Create gRPC server
	server := grpcpkg.NewServer(reg, eventBusAdapter)
	serverAddr := daemonConfig.GetServerAddress()

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		if serveForeground {
			fmt.Println("\n🛑 Shutting down gRPC server...")
		}
		cancel()
	}()

	// Log startup information
	if !serveForeground {
		// Background mode - just log
		logger.InfoContext(ctx, "Guild daemon starting",
			"display_name", daemonConfig.GetDisplayName(),
			"server_addr", serverAddr,
			"campaign", campaignName,
			"session", sessionNum,
			"log_file", daemonConfig.LogFile,
		)
		fmt.Printf("🏰 Guild server started in background\n")
		fmt.Printf("📋 Campaign: %s (session %d)\n", campaignName, sessionNum)
		fmt.Printf("🔌 Socket: %s\n", daemonConfig.SocketPath)
		fmt.Printf("💡 Use 'guild chat' to connect\n")
	} else {
		// Foreground mode - show detailed output
		if serveDev {
			fmt.Printf("🔧 %s running in DEVELOPMENT MODE at %s\n", daemonConfig.GetDisplayName(), serverAddr)
		} else {
			fmt.Printf("🏰 %s running at %s\n", daemonConfig.GetDisplayName(), serverAddr)
		}
		fmt.Printf("📋 Campaign: %s (session %d)\n", campaignName, sessionNum)
		fmt.Println("📡 Registered gRPC services:")
		fmt.Println("   ✅ Guild Service (campaigns, agents, commissions)")
		fmt.Println("   ✅ Chat Service (interactive agent communication)")
		fmt.Println("   ✅ Session Service (persistent session storage)")
		fmt.Println("   ✅ Event Service (real-time event streaming)")
		fmt.Println("   ✅ Prompt Service (prompt management)")
		fmt.Println()
		fmt.Printf("🔌 Socket: %s\n", daemonConfig.SocketPath)
		if serveConfig != "" {
			fmt.Printf("📄 Config: %s\n", serveConfig)
		}
		fmt.Println("💡 Use 'guild chat --campaign", campaignName, "' to connect")
		fmt.Println("🛑 Press Ctrl+C to stop the server")
	}

	// Start HTTP health endpoint if configured
	var healthServer *http.Server
	if yamlConfig, err := daemon.LoadYAMLConfig(serveConfig); err == nil && yamlConfig.Health.HTTPEnabled {
		healthServer = &http.Server{
			Addr: fmt.Sprintf(":%d", yamlConfig.Health.HTTPPort),
		}

		// Set up health endpoint
		http.HandleFunc(yamlConfig.Health.HTTPPath, func(w http.ResponseWriter, r *http.Request) {
			// TODO: Add actual health checks
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"status":"healthy","campaign":"%s","session":%d}`, campaignName, sessionNum)
		})

		// Start health server in background
		go func() {
			logger.InfoContext(ctx, "Starting HTTP health endpoint",
				"port", yamlConfig.Health.HTTPPort,
				"path", yamlConfig.Health.HTTPPath)
			if err := healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.ErrorContext(ctx, "Health endpoint failed", "error", err)
			}
		}()

		// Ensure health server shuts down on exit
		defer func() {
			if healthServer != nil {
				healthCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				healthServer.Shutdown(healthCtx)
			}
		}()
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

	// If running in background mode, fork and return
	if !serveForeground {
		// Fork the process to run in background
		if err := forkDaemon(daemonConfig); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to start background daemon").
				WithComponent("cli").
				WithOperation("serve.run")
		}
		// Parent process returns successfully
		return nil
	}

	// In foreground mode, start the server directly
	// Ensure cleanup on exit
	defer func() {
		// Shutdown event bus
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := unifiedEventBus.Close(shutdownCtx); err != nil {
			logger.ErrorContext(ctx, "Failed to shutdown event bus", "error", err)
		}

		// Clean up socket file
		if err := os.Remove(daemonConfig.SocketPath); err != nil && !os.IsNotExist(err) {
			logger.ErrorContext(ctx, "Failed to remove socket file", "socket", daemonConfig.SocketPath, "error", err)
		}
	}()

	startErr := server.StartUnix(ctx, daemonConfig.SocketPath)

	if startErr != nil {
		return gerror.Wrap(startErr, gerror.ErrCodeConnection, "failed to start gRPC server").
			WithComponent("cli").
			WithOperation("serve.run").
			WithDetails("server_address", serverAddr)
	}

	if serveForeground {
		fmt.Printf("✨ %s stopped gracefully...done.\n", daemonConfig.GetDisplayName())
	} else {
		logger.InfoContext(ctx, "Guild daemon stopped gracefully", "display_name", daemonConfig.GetDisplayName())
	}
	return nil
}

// forkDaemon starts the guild serve process in the background
func forkDaemon(daemonConfig *daemon.DaemonConfig) error {
	// Get the current executable path
	executable, err := os.Executable()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get executable path").
			WithComponent("cli").
			WithOperation("forkDaemon")
	}

	// Build the command arguments for the child process
	args := []string{"serve", "--foreground"}

	// Add the campaign flag if specified
	if serveCampaign != "" {
		args = append(args, "--campaign", serveCampaign)
	}

	// Add the session flag if specified
	if serveSession != "0" {
		args = append(args, "--session", serveSession)
	}

	// Add the socket flag if specified
	if serveSocket != "" {
		args = append(args, "--socket", serveSocket)
	}

	// Create the command
	cmd := exec.Command(executable, args...)

	// Detach from parent process
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}

	// Redirect output to log file
	logFile, err := os.OpenFile(daemonConfig.LogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to open log file").
			WithComponent("cli").
			WithOperation("forkDaemon").
			WithDetails("log_file", daemonConfig.LogFile)
	}
	defer logFile.Close()

	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// Start the process
	if err := cmd.Start(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to start daemon process").
			WithComponent("cli").
			WithOperation("forkDaemon")
	}

	// Write PID file
	pidFile := filepath.Join(filepath.Dir(daemonConfig.SocketPath), fmt.Sprintf("guild-%s.pid", daemonConfig.Campaign))
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0644); err != nil {
		// Non-fatal error - just log it
		fmt.Printf("⚠️  Warning: Could not write PID file: %v\n", err)
	}

	fmt.Printf("✅ Guild daemon started (PID: %d)\n", cmd.Process.Pid)

	// Release the process so it continues running after parent exits
	cmd.Process.Release()

	return nil
}
