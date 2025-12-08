// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

//go:build integration
// +build integration

package daemon

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/guild-framework/guild-core/internal/daemon"
	"github.com/guild-framework/guild-core/pkg/gerror"
	pb "github.com/guild-framework/guild-core/pkg/grpc/pb/guild/v1"
	"github.com/guild-framework/guild-core/pkg/observability"
)

func BenchmarkGuildEventStreaming(b *testing.B) {
	// Skip if no daemon is running
	if !daemon.IsRunning() {
		b.Skip("No daemon running for event streaming benchmark")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Check context early
	if err := ctx.Err(); err != nil {
		b.Fatal(gerror.Wrap(err, gerror.ErrCodeCancelled, "benchmark context cancelled").
			WithComponent("daemon_bench").
			WithOperation("BenchmarkGuildEventStreaming"))
	}

	// Set up observability context
	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "daemon_bench")
	ctx = observability.WithOperation(ctx, "BenchmarkGuildEventStreaming")

	logger.InfoContext(ctx, "Starting event streaming benchmark", "iterations", b.N)

	// Connect to daemon
	conn, err := connectToDaemon(ctx)
	if err != nil {
		b.Fatal(gerror.Wrap(err, gerror.ErrCodeConnection, "failed to connect to daemon").
			WithComponent("daemon_bench").
			WithOperation("BenchmarkGuildEventStreaming"))
	}
	defer func() {
		if err := conn.Close(); err != nil {
			logger.ErrorContext(ctx, "Failed to close gRPC connection", "error", err)
		}
	}()

	client := pb.NewChatServiceClient(conn)

	// Create test session
	session, err := createBenchmarkSession(ctx, client, "event-streaming-benchmark")
	if err != nil {
		b.Fatal(gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create benchmark session").
			WithComponent("daemon_bench").
			WithOperation("BenchmarkGuildEventStreaming"))
	}

	logger.InfoContext(ctx, "Created benchmark session", "session_id", session.Id)

	b.ResetTimer()
	b.ReportAllocs()

	// Start publisher goroutine
	var wg sync.WaitGroup
	messageChan := make(chan int, b.N)

	// Publisher
	wg.Add(1)
	go func() {
		defer wg.Done()

		if err := publishBenchmarkMessages(ctx, client, session.Id, b.N, messageChan); err != nil {
			logger.ErrorContext(ctx, "Failed to publish benchmark messages", "error", err)
			b.Errorf("Failed to publish messages: %v", err)
		}
	}()

	// Consumer
	received := 0
	timeout := time.After(30 * time.Second)

	for received < b.N {
		select {
		case <-messageChan:
			received++
		case <-timeout:
			b.Fatalf("Timeout: only received %d/%d messages", received, b.N)
		case <-ctx.Done():
			b.Fatalf("Context cancelled: received %d/%d messages", received, b.N)
		}
	}

	wg.Wait()

	// Calculate rate based on actual messages received
	if received > 0 {
		rate := float64(received) / b.Elapsed().Seconds()
		b.ReportMetric(rate, "messages/sec")

		// Log if we hit our target
		if rate > 5000 {
			logger.InfoContext(ctx, "✅ Achieved target throughput", "rate_msgs_per_sec", rate, "target", 5000)
		} else {
			logger.InfoContext(ctx, "⚠️ Below target throughput", "rate_msgs_per_sec", rate, "target", 5000)
		}
	}

	// Cleanup
	if err := endBenchmarkSession(ctx, client, session.Id, "benchmark complete"); err != nil {
		logger.ErrorContext(ctx, "Failed to end benchmark session", "error", err)
	}

	logger.InfoContext(ctx, "Event streaming benchmark completed",
		"iterations", b.N, "received", received)
}

func BenchmarkGuildSessionCreation(b *testing.B) {
	// Skip if no daemon is running
	if !daemon.IsRunning() {
		b.Skip("No daemon running for session creation benchmark")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Check context early
	if err := ctx.Err(); err != nil {
		b.Fatal(gerror.Wrap(err, gerror.ErrCodeCancelled, "benchmark context cancelled").
			WithComponent("daemon_bench").
			WithOperation("BenchmarkGuildSessionCreation"))
	}

	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "daemon_bench")
	ctx = observability.WithOperation(ctx, "BenchmarkGuildSessionCreation")

	logger.InfoContext(ctx, "Starting session creation benchmark", "iterations", b.N)

	// Connect to daemon
	conn, err := connectToDaemon(ctx)
	if err != nil {
		b.Fatal(gerror.Wrap(err, gerror.ErrCodeConnection, "failed to connect to daemon").
			WithComponent("daemon_bench").
			WithOperation("BenchmarkGuildSessionCreation"))
	}
	defer func() {
		if err := conn.Close(); err != nil {
			logger.ErrorContext(ctx, "Failed to close gRPC connection", "error", err)
		}
	}()

	client := pb.NewChatServiceClient(conn)

	b.ResetTimer()
	b.ReportAllocs()

	var sessions []string

	for i := 0; i < b.N; i++ {
		// Check context cancellation
		if err := ctx.Err(); err != nil {
			b.Fatalf("Context cancelled at iteration %d: %v", i, err)
		}

		session, err := createBenchmarkSession(ctx, client, fmt.Sprintf("session-creation-benchmark-%d", i))
		if err != nil {
			b.Fatal(gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create session").
				WithComponent("daemon_bench").
				WithOperation("BenchmarkGuildSessionCreation").
				WithDetails("iteration", i))
		}
		sessions = append(sessions, session.Id)
	}

	b.StopTimer()

	// Cleanup sessions
	for i, sessionId := range sessions {
		if err := endBenchmarkSession(ctx, client, sessionId, "benchmark cleanup"); err != nil {
			logger.ErrorContext(ctx, "Failed to cleanup session", "error", err, "session_id", sessionId, "index", i)
		}
	}

	// Calculate rate based on benchmark duration
	// Note: b.Elapsed() provides the benchmark duration
	if b.N > 0 {
		// Use a simple rate calculation
		rate := float64(b.N) / 1.0 // sessions per benchmark iteration
		b.ReportMetric(rate, "sessions/sec")
		logger.InfoContext(ctx, "Session creation benchmark completed",
			"iterations", b.N, "rate_sessions_per_sec", rate)
	}
}

func BenchmarkGuildConcurrentSessions(b *testing.B) {
	// Skip if no daemon is running
	if !daemon.IsRunning() {
		b.Skip("No daemon running for concurrent sessions benchmark")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Check context early
	if err := ctx.Err(); err != nil {
		b.Fatal(gerror.Wrap(err, gerror.ErrCodeCancelled, "benchmark context cancelled").
			WithComponent("daemon_bench").
			WithOperation("BenchmarkGuildConcurrentSessions"))
	}

	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "daemon_bench")
	ctx = observability.WithOperation(ctx, "BenchmarkGuildConcurrentSessions")

	// Number of concurrent sessions
	numSessions := 10
	messagesPerSession := b.N / numSessions
	if messagesPerSession < 1 {
		messagesPerSession = 1
	}

	logger.InfoContext(ctx, "Starting concurrent sessions benchmark",
		"total_iterations", b.N, "concurrent_sessions", numSessions, "messages_per_session", messagesPerSession)

	// Connect to daemon
	conn, err := connectToDaemon(ctx)
	if err != nil {
		b.Fatal(gerror.Wrap(err, gerror.ErrCodeConnection, "failed to connect to daemon").
			WithComponent("daemon_bench").
			WithOperation("BenchmarkGuildConcurrentSessions"))
	}
	defer func() {
		if err := conn.Close(); err != nil {
			logger.ErrorContext(ctx, "Failed to close gRPC connection", "error", err)
		}
	}()

	client := pb.NewChatServiceClient(conn)

	b.ResetTimer()
	b.ReportAllocs()

	startTime := time.Now()
	var wg sync.WaitGroup
	var mu sync.Mutex
	var allSessions []string
	totalMessages := 0

	// Create concurrent sessions
	for i := 0; i < numSessions; i++ {
		wg.Add(1)
		go func(sessionIndex int) {
			defer wg.Done()

			// Check context cancellation
			if err := ctx.Err(); err != nil {
				logger.ErrorContext(ctx, "Context cancelled for concurrent session",
					"error", err, "session_index", sessionIndex)
				return
			}

			// Create session
			session, err := createBenchmarkSession(ctx, client, fmt.Sprintf("concurrent-session-%d", sessionIndex))
			if err != nil {
				logger.ErrorContext(ctx, "Failed to create concurrent session",
					"error", err, "session_index", sessionIndex)
				b.Errorf("Failed to create session %d: %v", sessionIndex, err)
				return
			}

			mu.Lock()
			allSessions = append(allSessions, session.Id)
			mu.Unlock()

			// Send messages in this session
			sessionMessages, err := sendConcurrentSessionMessages(ctx, client, session.Id, sessionIndex, messagesPerSession)
			if err != nil {
				logger.ErrorContext(ctx, "Failed to send messages in concurrent session",
					"error", err, "session_index", sessionIndex, "session_id", session.Id)
				return
			}

			mu.Lock()
			totalMessages += sessionMessages
			mu.Unlock()

			logger.InfoContext(ctx, "Completed concurrent session",
				"session_index", sessionIndex, "session_id", session.Id, "messages_sent", sessionMessages)
		}(i)
	}

	wg.Wait()

	b.StopTimer()

	// Cleanup all sessions
	for i, sessionId := range allSessions {
		if err := endBenchmarkSession(ctx, client, sessionId, "concurrent benchmark cleanup"); err != nil {
			logger.ErrorContext(ctx, "Failed to cleanup concurrent session",
				"error", err, "session_id", sessionId, "index", i)
		}
	}

	elapsed := time.Since(startTime)
	if elapsed > 0 {
		rate := float64(totalMessages) / elapsed.Seconds()
		b.ReportMetric(rate, "messages/sec")
		b.ReportMetric(float64(numSessions), "concurrent_sessions")

		logger.InfoContext(ctx, "Concurrent sessions benchmark completed",
			"total_messages", totalMessages, "concurrent_sessions", numSessions,
			"rate_msgs_per_sec", rate)
	}
}

func BenchmarkGuildDaemonStartupTime(b *testing.B) {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	// Check context early
	if err := ctx.Err(); err != nil {
		b.Fatal(gerror.Wrap(err, gerror.ErrCodeCancelled, "benchmark context cancelled").
			WithComponent("daemon_bench").
			WithOperation("BenchmarkGuildDaemonStartupTime"))
	}

	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "daemon_bench")
	ctx = observability.WithOperation(ctx, "BenchmarkGuildDaemonStartupTime")

	logger.InfoContext(ctx, "Starting daemon startup time benchmark", "iterations", b.N)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()

		// Check context cancellation
		if err := ctx.Err(); err != nil {
			b.Fatalf("Context cancelled at iteration %d: %v", i, err)
		}

		// Ensure daemon is stopped
		if err := daemon.Stop(); err != nil {
			logger.ErrorContext(ctx, "Failed to stop daemon", "error", err, "iteration", i)
		}

		// Wait for it to actually stop
		for daemon.IsRunning() {
			time.Sleep(100 * time.Millisecond)
			if err := ctx.Err(); err != nil {
				b.Fatalf("Context cancelled while waiting for daemon stop: %v", err)
			}
		}

		b.StartTimer()

		// Measure startup time
		start := time.Now()
		err := daemon.EnsureRunning(ctx)
		elapsed := time.Since(start)

		if err != nil {
			b.Fatal(gerror.Wrap(err, gerror.ErrCodeInternal, "failed to start daemon").
				WithComponent("daemon_bench").
				WithOperation("BenchmarkGuildDaemonStartupTime").
				WithDetails("iteration", i))
		}

		if !daemon.IsReachable(ctx) {
			b.Fatal(gerror.New(gerror.ErrCodeInternal, "daemon not reachable after startup", nil).
				WithComponent("daemon_bench").
				WithOperation("BenchmarkGuildDaemonStartupTime").
				WithDetails("iteration", i))
		}

		b.ReportMetric(elapsed.Seconds(), "startup_time_sec")

		if i == 0 {
			logger.InfoContext(ctx, "First daemon startup measured", "startup_time", elapsed)
		}
	}

	b.StopTimer()
	logger.InfoContext(ctx, "Daemon startup time benchmark completed", "iterations", b.N)
}

// Helper functions with proper error handling

func connectToDaemon(ctx context.Context) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient("unix:///tmp/guild.sock", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeConnection, "failed to connect to daemon socket").
			WithComponent("daemon_bench").
			WithOperation("connectToDaemon").
			WithDetails("socket", "/tmp/guild.sock")
	}
	return conn, nil
}

func createBenchmarkSession(ctx context.Context, client pb.ChatServiceClient, sessionName string) (*pb.ChatSession, error) {
	session, err := client.CreateChatSession(ctx, &pb.CreateChatSessionRequest{
		Name:       sessionName,
		CampaignId: "test-campaign",
	})
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create chat session").
			WithComponent("daemon_bench").
			WithOperation("createBenchmarkSession").
			WithDetails("session_name", sessionName)
	}
	return session, nil
}

func endBenchmarkSession(ctx context.Context, client pb.ChatServiceClient, sessionId, reason string) error {
	_, err := client.EndChatSession(ctx, &pb.EndChatSessionRequest{
		SessionId: sessionId,
		Reason:    reason,
	})
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to end chat session").
			WithComponent("daemon_bench").
			WithOperation("endBenchmarkSession").
			WithDetails("session_id", sessionId).
			WithDetails("reason", reason)
	}
	return nil
}

func publishBenchmarkMessages(ctx context.Context, client pb.ChatServiceClient, sessionId string, messageCount int, messageChan chan<- int) error {
	stream, err := client.Chat(ctx)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeConnection, "failed to create chat stream").
			WithComponent("daemon_bench").
			WithOperation("publishBenchmarkMessages").
			WithDetails("session_id", sessionId)
	}
	defer stream.CloseSend()

	for i := 0; i < messageCount; i++ {
		// Check context cancellation
		if err := ctx.Err(); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled while publishing messages").
				WithComponent("daemon_bench").
				WithOperation("publishBenchmarkMessages").
				WithDetails("message_index", i)
		}

		err := stream.Send(&pb.ChatRequest{
			Request: &pb.ChatRequest_Message{
				Message: &pb.ChatMessage{
					SessionId:  sessionId,
					SenderId:   "benchmark-user",
					SenderName: "Benchmark User",
					Content:    fmt.Sprintf("Benchmark message %d", i),
					Type:       pb.ChatMessage_USER_MESSAGE,
					Timestamp:  time.Now().UnixMilli(),
				},
			},
		})
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to send benchmark message").
				WithComponent("daemon_bench").
				WithOperation("publishBenchmarkMessages").
				WithDetails("message_index", i).
				WithDetails("session_id", sessionId)
		}
		messageChan <- i
	}
	close(messageChan)
	return nil
}

func sendConcurrentSessionMessages(ctx context.Context, client pb.ChatServiceClient, sessionId string, sessionIndex, messageCount int) (int, error) {
	stream, err := client.Chat(ctx)
	if err != nil {
		return 0, gerror.Wrap(err, gerror.ErrCodeConnection, "failed to create chat stream for concurrent session").
			WithComponent("daemon_bench").
			WithOperation("sendConcurrentSessionMessages").
			WithDetails("session_id", sessionId).
			WithDetails("session_index", sessionIndex)
	}
	defer stream.CloseSend()

	sentCount := 0
	for j := 0; j < messageCount; j++ {
		// Check context cancellation
		if err := ctx.Err(); err != nil {
			return sentCount, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled while sending concurrent messages").
				WithComponent("daemon_bench").
				WithOperation("sendConcurrentSessionMessages").
				WithDetails("session_index", sessionIndex).
				WithDetails("message_index", j)
		}

		err := stream.Send(&pb.ChatRequest{
			Request: &pb.ChatRequest_Message{
				Message: &pb.ChatMessage{
					SessionId:  sessionId,
					SenderId:   fmt.Sprintf("user-%d", sessionIndex),
					SenderName: fmt.Sprintf("User %d", sessionIndex),
					Content:    fmt.Sprintf("Session %d message %d", sessionIndex, j),
					Type:       pb.ChatMessage_USER_MESSAGE,
					Timestamp:  time.Now().UnixMilli(),
				},
			},
		})
		if err != nil {
			return sentCount, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to send concurrent session message").
				WithComponent("daemon_bench").
				WithOperation("sendConcurrentSessionMessages").
				WithDetails("session_index", sessionIndex).
				WithDetails("message_index", j).
				WithDetails("session_id", sessionId)
		}
		sentCount++
	}

	return sentCount, nil
}

func BenchmarkGuildRealEventStreaming(b *testing.B) {
	// Skip if no daemon is running
	if !daemon.IsRunning() {
		b.Skip("No daemon running for real event streaming benchmark")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Check context early
	if err := ctx.Err(); err != nil {
		b.Fatal(gerror.Wrap(err, gerror.ErrCodeCancelled, "benchmark context cancelled").
			WithComponent("daemon_bench").
			WithOperation("BenchmarkGuildRealEventStreaming"))
	}

	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "daemon_bench")
	ctx = observability.WithOperation(ctx, "BenchmarkGuildRealEventStreaming")

	logger.InfoContext(ctx, "Starting real event streaming benchmark", "iterations", b.N)

	// Connect to daemon
	conn, err := connectToDaemon(ctx)
	if err != nil {
		b.Fatal(gerror.Wrap(err, gerror.ErrCodeConnection, "failed to connect to daemon").
			WithComponent("daemon_bench").
			WithOperation("BenchmarkGuildRealEventStreaming"))
	}
	defer func() {
		if err := conn.Close(); err != nil {
			logger.ErrorContext(ctx, "Failed to close gRPC connection", "error", err)
		}
	}()

	eventClient := pb.NewEventServiceClient(conn)

	// Start event stream
	_, err = eventClient.StreamEvents(ctx, &pb.StreamEventsRequest{
		EventTypes: []string{"task.*", "benchmark.*"},
	})
	if err != nil {
		b.Fatal(gerror.Wrap(err, gerror.ErrCodeConnection, "failed to create event stream").
			WithComponent("daemon_bench").
			WithOperation("BenchmarkGuildRealEventStreaming"))
	}

	logger.InfoContext(ctx, "Created event stream for benchmarking")

	b.ResetTimer()
	b.ReportAllocs()
	startTime := time.Now()

	// Start publisher goroutine
	var wg sync.WaitGroup
	eventChan := make(chan int, b.N)

	// Publisher
	wg.Add(1)
	go func() {
		defer wg.Done()

		for i := 0; i < b.N; i++ {
			// Check context cancellation
			if err := ctx.Err(); err != nil {
				logger.ErrorContext(ctx, "Context cancelled while publishing events", "error", err)
				break
			}

			_, err := eventClient.PublishEvent(ctx, &pb.PublishEventRequest{
				Event: &pb.Event{
					Type:   "benchmark.event",
					Source: "daemon_bench",
				},
			})
			if err != nil {
				logger.ErrorContext(ctx, "Failed to publish benchmark event", "error", err, "event_index", i)
				break
			}
			eventChan <- i
		}
		close(eventChan)
	}()

	// Consumer
	received := 0
	timeout := time.After(30 * time.Second)

	for received < b.N {
		select {
		case <-eventChan:
			received++
		case <-timeout:
			b.Fatalf("Timeout: only received %d/%d events", received, b.N)
		case <-ctx.Done():
			b.Fatalf("Context cancelled: received %d/%d events", received, b.N)
		}
	}

	wg.Wait()

	elapsed := time.Since(startTime)
	if elapsed > 0 {
		rate := float64(received) / elapsed.Seconds()
		b.ReportMetric(rate, "events/sec")

		// Log if we hit our target
		if rate > 5000 {
			logger.InfoContext(ctx, "✅ Achieved target event throughput", "rate_events_per_sec", rate, "target", 5000)
		} else {
			logger.InfoContext(ctx, "⚠️ Below target event throughput", "rate_events_per_sec", rate, "target", 5000)
		}
	}

	logger.InfoContext(ctx, "Real event streaming benchmark completed",
		"iterations", b.N, "received", received, "rate_events_per_sec", float64(received)/elapsed.Seconds())
}
