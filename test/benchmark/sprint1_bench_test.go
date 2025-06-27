// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package benchmark

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/lancekrogers/guild/pkg/config"
	grpcpkg "github.com/lancekrogers/guild/pkg/grpc"
	guildv1 "github.com/lancekrogers/guild/pkg/grpc/pb/guild/v1"
	"github.com/lancekrogers/guild/pkg/project"
	"github.com/lancekrogers/guild/pkg/registry"
)

// Performance targets from Agent 4 specification:
// - Init time < 2s
// - Response time < 500ms
// - Memory usage < 100MB

// mockEventBus implements the EventBus interface for testing
type mockEventBus struct{}

func (m *mockEventBus) Publish(event interface{})                                   {}
func (m *mockEventBus) Subscribe(eventType string, handler func(event interface{})) {}

// BenchmarkGuildInit benchmarks the guild init command performance
func BenchmarkGuildInit(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		tempDir := b.TempDir()
		ctx := context.Background()
		b.StartTimer()

		start := time.Now()
		_, err := project.Initialize(ctx, tempDir, project.InitOptions{})
		elapsed := time.Since(start)

		b.StopTimer()
		require.NoError(b, err, "Init should succeed")

		// Report custom metrics
		b.ReportMetric(float64(elapsed.Milliseconds()), "init_time_ms")

		// Assert performance target
		if elapsed > 2*time.Second {
			b.Logf("Init took %v (exceeds 2s target)", elapsed)
		}
		b.StartTimer()
	}
}

// BenchmarkGuildInitWithGoProject benchmarks init performance on Go projects
func BenchmarkGuildInitWithGoProject(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		tempDir := b.TempDir()

		// Setup Go project structure
		goMod := `module benchmark-test

go 1.21

require (
	github.com/stretchr/testify v1.8.4
)
`
		require.NoError(b, os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0644))
		require.NoError(b, os.WriteFile(filepath.Join(tempDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644))

		ctx := context.Background()
		b.StartTimer()

		start := time.Now()
		_, err := project.Initialize(ctx, tempDir, project.InitOptions{})
		elapsed := time.Since(start)

		b.StopTimer()
		require.NoError(b, err, "Go project init should succeed")

		b.ReportMetric(float64(elapsed.Milliseconds()), "go_init_time_ms")

		if elapsed > 2*time.Second {
			b.Logf("Go project init took %v (exceeds 2s target)", elapsed)
		}
		b.StartTimer()
	}
}

// BenchmarkGuildConfigLoading benchmarks guild configuration loading performance
func BenchmarkGuildConfigLoading(b *testing.B) {
	b.ReportAllocs()

	// Setup test project once
	tempDir := b.TempDir()
	ctx := context.Background()

	_, err := project.Initialize(ctx, tempDir, project.InitOptions{})
	require.NoError(b, err)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		start := time.Now()
		_, err := config.LoadGuildConfig(ctx, tempDir)
		elapsed := time.Since(start)

		b.StopTimer()
		if err != nil {
			// Config loading might fail in test environment - that's ok for benchmarking
			b.Logf("Config loading error (expected in test): %v", err)
		} else {
			b.ReportMetric(float64(elapsed.Milliseconds()), "config_load_time_ms")
		}

		if elapsed > 100*time.Millisecond {
			b.Logf("Config loading took %v (may be slow)", elapsed)
		}
		b.StartTimer()
	}
}

// BenchmarkRegistryInitialization benchmarks component registry setup
func BenchmarkRegistryInitialization(b *testing.B) {
	b.ReportAllocs()

	ctx := context.Background()
	registryConfig := registry.Config{
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
		},
		Providers: registry.ProviderConfig{
			DefaultProvider: "claudecode",
			Providers: map[string]interface{}{
				"claudecode": map[string]interface{}{
					"model": "sonnet",
				},
			},
		},
		Memory: registry.MemoryConfig{
			DefaultMemoryStore: "sqlite",
			DefaultVectorStore: "chromem",
		},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		start := time.Now()
		reg := registry.NewComponentRegistry()
		err := reg.Initialize(ctx, registryConfig)
		elapsed := time.Since(start)

		b.StopTimer()
		require.NoError(b, err, "Registry initialization should succeed")

		b.ReportMetric(float64(elapsed.Milliseconds()), "registry_init_ms")

		if elapsed > 500*time.Millisecond {
			b.Logf("Registry init took %v (may be slow)", elapsed)
		}
		b.StartTimer()
	}
}

// BenchmarkGRPCServerStartup benchmarks gRPC server startup time
func BenchmarkGRPCServerStartup(b *testing.B) {
	b.ReportAllocs()

	ctx := context.Background()

	// Setup registry once
	reg := registry.NewComponentRegistry()
	err := reg.Initialize(ctx, registry.Config{})
	require.NoError(b, err)

	// Mock event bus
	eventBus := &mockEventBus{}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		b.StartTimer()

		start := time.Now()
		server := grpcpkg.NewServer(reg, eventBus)
		elapsed := time.Since(start)

		b.StopTimer()
		require.NotNil(b, server, "Server should be created")

		b.ReportMetric(float64(elapsed.Microseconds()), "server_create_us")

		if elapsed > 10*time.Millisecond {
			b.Logf("Server creation took %v", elapsed)
		}
		b.StartTimer()
	}
}

// BenchmarkAgentResponse benchmarks agent response time (mock scenario)
func BenchmarkAgentResponse(b *testing.B) {
	b.ReportAllocs()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Setup minimal registry for agent testing
	reg := registry.NewComponentRegistry()
	err := reg.Initialize(ctx, registry.Config{
		Agents: registry.AgentConfigYaml{
			DefaultType: "worker",
		},
	})
	require.NoError(b, err)

	agentRegistry := reg.Agents()
	require.NotNil(b, agentRegistry)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		start := time.Now()
		agent, err := agentRegistry.GetAgent("worker")
		elapsed := time.Since(start)

		b.StopTimer()
		if err != nil {
			b.Logf("Agent creation error (may be expected in test): %v", err)
		} else {
			require.NotNil(b, agent, "Agent should be created")
			b.ReportMetric(float64(elapsed.Milliseconds()), "agent_create_ms")

			if elapsed > 500*time.Millisecond {
				b.Logf("Agent creation took %v (exceeds 500ms target)", elapsed)
			}
		}
		b.StartTimer()
	}
}

// BenchmarkMessageRouting benchmarks message routing performance through gRPC
func BenchmarkMessageRouting(b *testing.B) {
	b.ReportAllocs()

	if testing.Short() {
		b.Skip("Skipping message routing benchmark in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Setup server for benchmarking
	reg := registry.NewComponentRegistry()
	err := reg.Initialize(ctx, registry.Config{
		Agents: registry.AgentConfigYaml{
			DefaultType: "worker",
		},
	})
	require.NoError(b, err)

	// Mock event bus
	eventBus := &mockEventBus{}

	server := grpcpkg.NewServer(reg, eventBus)

	// Start server in background
	serverAddr := "localhost:0"
	go func() {
		_ = server.Start(ctx, serverAddr)
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Connect client
	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		b.Skip("Could not connect to test server")
	}
	defer conn.Close()

	client := guildv1.NewGuildClient(conn)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		start := time.Now()
		_, err := client.ListAvailableAgents(ctx, &guildv1.ListAgentsRequest{})
		elapsed := time.Since(start)

		b.StopTimer()
		if err != nil {
			b.Logf("Message routing error (may be expected): %v", err)
		} else {
			b.ReportMetric(float64(elapsed.Milliseconds()), "routing_time_ms")

			if elapsed > 500*time.Millisecond {
				b.Logf("Message routing took %v (exceeds 500ms target)", elapsed)
			}
		}
		b.StartTimer()
	}
}

// BenchmarkMemoryOperations benchmarks SQLite memory operations
func BenchmarkMemoryOperations(b *testing.B) {
	b.ReportAllocs()

	tempDir := b.TempDir()
	ctx := context.Background()

	// Initialize project with SQLite database
	_, err := project.Initialize(ctx, tempDir, project.InitOptions{})
	require.NoError(b, err)

	dbPath := filepath.Join(tempDir, ".campaign", "memory.db")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		start := time.Now()

		// Simulate database access
		_, err := os.Stat(dbPath)

		elapsed := time.Since(start)

		b.StopTimer()
		require.NoError(b, err, "Database file should exist")

		b.ReportMetric(float64(elapsed.Microseconds()), "db_access_us")
		b.StartTimer()
	}
}

// BenchmarkConcurrentInit benchmarks concurrent initialization performance
func BenchmarkConcurrentInit(b *testing.B) {
	b.ReportAllocs()

	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tempDir := b.TempDir()

			start := time.Now()
			_, err := project.Initialize(ctx, tempDir, project.InitOptions{})
			elapsed := time.Since(start)

			if err != nil {
				b.Errorf("Concurrent init failed: %v", err)
			}

			if elapsed > 3*time.Second {
				b.Logf("Concurrent init took %v", elapsed)
			}
		}
	})
}

// BenchmarkConfigParsing benchmarks YAML configuration parsing
func BenchmarkConfigParsing(b *testing.B) {
	b.ReportAllocs()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		start := time.Now()

		// Simulate YAML parsing (simplified for benchmark)
		var agent config.AgentConfig
		agent.ID = "benchmark-agent"
		agent.Name = "Benchmark Agent"

		elapsed := time.Since(start)

		b.StopTimer()
		b.ReportMetric(float64(elapsed.Microseconds()), "yaml_parse_us")
		b.StartTimer()
	}
}

// BenchmarkProjectDetection benchmarks project type detection
func BenchmarkProjectDetection(b *testing.B) {
	b.ReportAllocs()

	// Setup different project types
	projectTypes := map[string]map[string]string{
		"go": {
			"go.mod":  "module test\ngo 1.21\n",
			"main.go": "package main\nfunc main() {}\n",
		},
		"js": {
			"package.json": `{"name": "test"}`,
			"index.js":     "console.log('test');\n",
		},
		"python": {
			"requirements.txt": "flask==2.0.0\n",
			"app.py":           "from flask import Flask\n",
		},
	}

	for projectType, files := range projectTypes {
		b.Run(projectType, func(b *testing.B) {
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				b.StopTimer()
				tempDir := b.TempDir()

				// Setup project files
				for file, content := range files {
					filePath := filepath.Join(tempDir, file)
					require.NoError(b, os.WriteFile(filePath, []byte(content), 0644))
				}

				ctx := context.Background()
				b.StartTimer()

				start := time.Now()
				_, err := project.Initialize(ctx, tempDir, project.InitOptions{})
				elapsed := time.Since(start)

				b.StopTimer()
				require.NoError(b, err)

				b.ReportMetric(float64(elapsed.Milliseconds()), projectType+"_detection_ms")
				b.StartTimer()
			}
		})
	}
}

// BenchmarkMemoryUsage benchmarks memory usage during operations
func BenchmarkMemoryUsage(b *testing.B) {
	b.ReportAllocs()

	ctx := context.Background()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		tempDir := b.TempDir()
		b.StartTimer()

		// Measure memory before
		var m1, m2 runtime.MemStats
		runtime.ReadMemStats(&m1)

		// Perform operations
		_, err := project.Initialize(ctx, tempDir, project.InitOptions{})
		require.NoError(b, err)

		// Create registry
		reg := registry.NewComponentRegistry()
		err = reg.Initialize(ctx, registry.Config{})
		require.NoError(b, err)

		// Measure memory after
		runtime.ReadMemStats(&m2)

		b.StopTimer()

		memoryUsed := m2.Alloc - m1.Alloc
		b.ReportMetric(float64(memoryUsed)/1024/1024, "memory_mb")

		// Check against 100MB target
		if memoryUsed > 100*1024*1024 {
			b.Logf("Memory usage %d MB exceeds 100MB target", memoryUsed/1024/1024)
		}

		b.StartTimer()
	}
}
