// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package infrastructure

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/guild-ventures/guild-core/internal/testutil"
	grpcserver "github.com/guild-ventures/guild-core/pkg/grpc"
	pb "github.com/guild-ventures/guild-core/pkg/grpc/pb/guild/v1"
	"github.com/guild-ventures/guild-core/pkg/registry"
)

// Simple in-memory event bus for testing
type inMemoryEventBus struct {
	handlers map[string][]func(event interface{})
	mu       sync.RWMutex
}

func (b *inMemoryEventBus) Publish(event interface{}) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Get event type name
	eventType := fmt.Sprintf("%T", event)

	// Call all handlers for this event type
	for _, handler := range b.handlers[eventType] {
		handler(event)
	}
}

func (b *inMemoryEventBus) Subscribe(eventType string, handler func(event interface{})) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

// TestBidirectionalStreamingUnderLoad tests gRPC streaming with high message volume
// TODO: This test needs major rework to properly handle the chat service requirements:
//  1. The ChatService expects sessions to be created via CreateChatSession before messages can be sent
//  2. The test should either:
//     a) Create sessions properly before sending messages
//     b) Use a simpler mock service that doesn't require session management
//     c) Focus on pure gRPC infrastructure testing without the chat logic
//  3. The current implementation causes session not found errors and times out
//  4. Consider creating a separate SimpleStreamingService for infrastructure testing
func TestBidirectionalStreamingUnderLoad(t *testing.T) {
	t.Skip("Test needs rework to handle session creation requirements")
	ctx := context.Background()

	// Setup server
	server, addr := setupTestGRPCServer(t)
	defer server.Stop()

	// Create client connection
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	require.NoError(t, err)
	defer conn.Close()

	client := pb.NewChatServiceClient(conn)

	t.Run("1000ConcurrentMessages", func(t *testing.T) {
		stream, err := client.Chat(ctx)
		require.NoError(t, err)

		// Message counters
		var sentCount, receivedCount int64
		messageCount := 1000

		// Start receiver goroutine
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				msg, err := stream.Recv()
				if err != nil {
					return
				}
				if msg != nil {
					atomic.AddInt64(&receivedCount, 1)
				}
			}
		}()

		// Send messages concurrently
		startTime := time.Now()
		sendWg := sync.WaitGroup{}
		for i := 0; i < messageCount; i++ {
			sendWg.Add(1)
			go func(msgNum int) {
				defer sendWg.Done()
				err := stream.Send(&pb.ChatRequest{
					Request: &pb.ChatRequest_Message{
						Message: &pb.ChatMessage{
							SessionId: "test-session",
							SenderId:  "user",
							Content:   fmt.Sprintf("Test message %d", msgNum),
							Timestamp: time.Now().Unix(),
						},
					},
				})
				if err == nil {
					atomic.AddInt64(&sentCount, 1)
				}
			}(i)
		}

		// Wait for sends to complete
		sendWg.Wait()
		duration := time.Since(startTime)

		// Give receiver time to catch up
		time.Sleep(100 * time.Millisecond)
		stream.CloseSend()
		wg.Wait()

		// Verify results
		assert.Equal(t, int64(messageCount), sentCount, "All messages should be sent")
		assert.Greater(t, receivedCount, int64(900), "Should receive most messages (>90%)")
		assert.Less(t, duration, 5*time.Second, "Should handle 1000 messages quickly")

		// Calculate throughput
		throughput := float64(sentCount) / duration.Seconds()
		t.Logf("Throughput: %.2f messages/second", throughput)
		assert.Greater(t, throughput, 200.0, "Should handle >200 messages/second")
	})

	t.Run("MessageOrderingGuarantees", func(t *testing.T) {
		stream, err := client.Chat(ctx)
		require.NoError(t, err)

		// Track received message order
		receivedOrder := make([]int, 0)
		var mu sync.Mutex

		// Start receiver
		done := make(chan bool)
		go func() {
			for {
				msg, err := stream.Recv()
				if err != nil {
					close(done)
					return
				}

				// Extract message number from content
				var msgNum int
				if chatMsg := msg.GetMessage(); chatMsg != nil {
					fmt.Sscanf(chatMsg.Content, "Ordered message %d", &msgNum)

					mu.Lock()
					receivedOrder = append(receivedOrder, msgNum)
					mu.Unlock()
				}
			}
		}()

		// Send ordered messages
		for i := 0; i < 100; i++ {
			err := stream.Send(&pb.ChatRequest{
				Request: &pb.ChatRequest_Message{
					Message: &pb.ChatMessage{
						SessionId: "test-session",
						SenderId:  "user",
						Content:   fmt.Sprintf("Ordered message %d", i),
						Timestamp: time.Now().Unix(),
					},
				},
			})
			require.NoError(t, err)
		}

		// Close and wait
		stream.CloseSend()
		<-done

		// Verify order preserved
		mu.Lock()
		defer mu.Unlock()

		for i := 1; i < len(receivedOrder); i++ {
			assert.Greater(t, receivedOrder[i], receivedOrder[i-1],
				"Messages should be received in order")
		}
	})

	t.Run("BackpressureHandling", func(t *testing.T) {
		stream, err := client.Chat(ctx)
		require.NoError(t, err)

		// Slow receiver to create backpressure
		slowReceiver := make(chan *pb.ChatResponse, 10)
		go func() {
			for {
				msg, err := stream.Recv()
				if err != nil {
					close(slowReceiver)
					return
				}

				// Simulate slow processing
				time.Sleep(10 * time.Millisecond)
				select {
				case slowReceiver <- msg:
				default:
					// Drop if buffer full
				}
			}
		}()

		// Send burst of messages
		sendErrors := 0
		for i := 0; i < 1000; i++ {
			err := stream.Send(&pb.ChatRequest{
				Request: &pb.ChatRequest_Message{
					Message: &pb.ChatMessage{
						SessionId: "test-session",
						SenderId:  "user",
						Content:   "Burst message",
						Timestamp: time.Now().Unix(),
					},
				},
			})
			if err != nil {
				sendErrors++
			}
		}

		stream.CloseSend()

		// Should handle backpressure gracefully
		assert.Less(t, sendErrors, 100, "Should have few send errors despite backpressure")
	})

	t.Run("MemoryUsageUnderLoad", func(t *testing.T) {
		// Get baseline memory
		var m runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m)
		baselineMemory := m.Alloc

		stream, err := client.Chat(ctx)
		require.NoError(t, err)

		// Send large messages
		largeContent := strings.Repeat("x", 10000) // 10KB per message
		for i := 0; i < 100; i++ {
			err := stream.Send(&pb.ChatRequest{
				Request: &pb.ChatRequest_Message{
					Message: &pb.ChatMessage{
						SessionId: "test-session",
						SenderId:  "user",
						Content:   largeContent,
						Timestamp: time.Now().Unix(),
					},
				},
			})
			require.NoError(t, err)
		}

		// Check memory after load
		runtime.GC()
		runtime.ReadMemStats(&m)
		memoryGrowth := m.Alloc - baselineMemory

		// Memory should not grow excessively
		maxExpectedGrowth := uint64(100 * 10000 * 2) // 100 messages * 10KB * 2 (for overhead)
		assert.Less(t, memoryGrowth, maxExpectedGrowth,
			"Memory growth should be bounded")
	})
}

// TestConnectionRecovery tests network failure scenarios and recovery
// TODO: This test also needs session management added
func TestConnectionRecovery(t *testing.T) {
	t.Skip("Test needs rework to handle session creation requirements")
	ctx := context.Background()

	t.Run("NetworkInterruptionRecovery", func(t *testing.T) {
		// Setup server with controlled listener
		lis, err := net.Listen("tcp", "localhost:0")
		require.NoError(t, err)

		// Wrap listener to control connections
		controlledLis := &controlledListener{
			Listener: lis,
			active:   true,
		}

		server := grpc.NewServer()
		chatService := createTestGRPCServer(t)
		pb.RegisterChatServiceServer(server, chatService)

		go server.Serve(controlledLis)
		defer server.Stop()

		// Create client with retry
		conn, err := grpc.Dial(
			controlledLis.Addr().String(),
			grpc.WithInsecure(),
			grpc.WithDefaultCallOptions(
				grpc.WaitForReady(true),
			),
		)
		require.NoError(t, err)
		defer conn.Close()

		client := pb.NewChatServiceClient(conn)

		// Start streaming
		stream, err := client.Chat(ctx)
		require.NoError(t, err)

		// Send initial messages
		for i := 0; i < 5; i++ {
			err := stream.Send(&pb.ChatRequest{
				Request: &pb.ChatRequest_Message{
					Message: &pb.ChatMessage{
						SessionId: "test-session",
						SenderId:  "user",
						Content:   "Before interruption",
						Timestamp: time.Now().Unix(),
					},
				},
			})
			require.NoError(t, err)
		}

		// Simulate network interruption
		controlledLis.SetActive(false)
		time.Sleep(100 * time.Millisecond)

		// Try to send during interruption (should fail)
		err = stream.Send(&pb.ChatRequest{
			Request: &pb.ChatRequest_Message{
				Message: &pb.ChatMessage{
					SessionId: "test-session",
					SenderId:  "user",
					Content:   "During interruption",
					Timestamp: time.Now().Unix(),
				},
			},
		})
		assert.Error(t, err, "Send should fail during interruption")

		// Restore connection
		controlledLis.SetActive(true)
		time.Sleep(100 * time.Millisecond)

		// Create new stream after recovery
		stream, err = client.Chat(ctx)
		require.NoError(t, err, "Should reconnect after network recovery")

		// Send post-recovery messages
		for i := 0; i < 5; i++ {
			err := stream.Send(&pb.ChatRequest{
				Request: &pb.ChatRequest_Message{
					Message: &pb.ChatMessage{
						SessionId: "test-session",
						SenderId:  "user",
						Content:   "After recovery",
						Timestamp: time.Now().Unix(),
					},
				},
			})
			require.NoError(t, err, "Should send after recovery")
		}
	})

	t.Run("AutomaticReconnection", func(t *testing.T) {
		// Server that tracks connections
		connectionCount := int32(0)

		server := grpc.NewServer(
			grpc.UnaryInterceptor(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
				atomic.AddInt32(&connectionCount, 1)
				return handler(ctx, req)
			}),
		)

		lis, err := net.Listen("tcp", "localhost:0")
		require.NoError(t, err)

		chatService := createTestGRPCServer(t)
		pb.RegisterChatServiceServer(server, chatService)

		go server.Serve(lis)
		defer server.Stop()

		// Client with aggressive retry
		conn, err := grpc.Dial(
			lis.Addr().String(),
			grpc.WithInsecure(),
			grpc.WithBackoffConfig(grpc.BackoffConfig{
				MaxDelay: 500 * time.Millisecond,
			}),
		)
		require.NoError(t, err)
		defer conn.Close()

		// Force reconnection by closing server
		server.Stop()
		time.Sleep(100 * time.Millisecond)

		// Restart server on same port
		newLis, err := net.Listen("tcp", lis.Addr().String())
		require.NoError(t, err)

		newServer := grpc.NewServer()
		pb.RegisterChatServiceServer(newServer, chatService)
		go newServer.Serve(newLis)
		defer newServer.Stop()

		// Client should automatically reconnect
		client := pb.NewChatServiceClient(conn)
		ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		// This should work after automatic reconnection
		stream, err := client.Chat(ctx)
		require.NoError(t, err, "Should reconnect automatically")

		err = stream.Send(&pb.ChatRequest{
			Request: &pb.ChatRequest_Message{
				Message: &pb.ChatMessage{
					SessionId: "test-session",
					SenderId:  "user",
					Content:   "After reconnection",
					Timestamp: time.Now().Unix(),
				},
			},
		})
		require.NoError(t, err, "Should send after reconnection")
	})

	t.Run("StateSynchronization", func(t *testing.T) {
		// Server that maintains state
		server, addr := setupTestGRPCServer(t)
		defer server.Stop()

		conn, err := grpc.Dial(addr, grpc.WithInsecure())
		require.NoError(t, err)
		defer conn.Close()

		client := pb.NewChatServiceClient(conn)

		// First session - establish state
		session1, err := client.Chat(ctx)
		require.NoError(t, err)

		// Send messages with session ID
		sessionID := "test-session-123"
		for i := 0; i < 5; i++ {
			err := session1.Send(&pb.ChatRequest{
				Request: &pb.ChatRequest_Message{
					Message: &pb.ChatMessage{
						SessionId: sessionID,
						SenderId:  "user",
						Content:   fmt.Sprintf("Message %d", i),
						Timestamp: time.Now().Unix(),
					},
				},
			})
			require.NoError(t, err)
		}
		session1.CloseSend()

		// Simulate disconnection/reconnection with new stream
		session2, err := client.Chat(ctx)
		require.NoError(t, err)

		// Request state sync
		err = session2.Send(&pb.ChatRequest{
			Request: &pb.ChatRequest_Control{
				Control: &pb.ChatControl{
					Action:    pb.ChatControl_REQUEST_STATUS,
					SessionId: sessionID,
				},
			},
		})
		require.NoError(t, err)

		// Should receive state sync response
		syncResp, err := session2.Recv()
		require.NoError(t, err)
		// Verify response is a valid chat event
		assert.NotNil(t, syncResp.GetEvent())

		// Verify we can continue from where we left off
		err = session2.Send(&pb.ChatRequest{
			Request: &pb.ChatRequest_Message{
				Message: &pb.ChatMessage{
					SessionId: sessionID,
					SenderId:  "user",
					Content:   "Continuing after sync",
					Timestamp: time.Now().Unix(),
				},
			},
		})
		require.NoError(t, err)
	})
}

// TestConcurrentClientHandling tests server behavior with many simultaneous clients
// TODO: This test also needs session management added
func TestConcurrentClientHandling(t *testing.T) {
	t.Skip("Test needs rework to handle session creation requirements")
	ctx := context.Background()
	server, addr := setupTestGRPCServer(t)
	defer server.Stop()

	t.Run("100SimultaneousClients", func(t *testing.T) {
		clientCount := 100
		messagesPerClient := 10

		var successfulClients int32
		var totalMessagesSent int32

		// Start clients concurrently
		var wg sync.WaitGroup
		startTime := time.Now()

		for i := 0; i < clientCount; i++ {
			wg.Add(1)
			go func(clientID int) {
				defer wg.Done()

				// Each client gets its own connection
				conn, err := grpc.Dial(addr, grpc.WithInsecure())
				if err != nil {
					return
				}
				defer conn.Close()

				client := pb.NewChatServiceClient(conn)
				stream, err := client.Chat(ctx)
				if err != nil {
					return
				}

				// Send and receive messages
				clientSuccess := true
				for j := 0; j < messagesPerClient; j++ {
					err := stream.Send(&pb.ChatRequest{
						Request: &pb.ChatRequest_Message{
							Message: &pb.ChatMessage{
								SessionId: fmt.Sprintf("client-%d", clientID),
								SenderId:  "user",
								Content:   fmt.Sprintf("Message from client %d", clientID),
								Timestamp: time.Now().Unix(),
							},
						},
					})
					if err != nil {
						clientSuccess = false
						break
					}
					atomic.AddInt32(&totalMessagesSent, 1)
				}

				if clientSuccess {
					atomic.AddInt32(&successfulClients, 1)
				}
				stream.CloseSend()
			}(i)
		}

		wg.Wait()
		duration := time.Since(startTime)

		// Verify results
		assert.Greater(t, successfulClients, int32(90), "At least 90% of clients should succeed")
		assert.Greater(t, totalMessagesSent, int32(900), "Should send most messages")
		assert.Less(t, duration, 10*time.Second, "Should handle 100 clients within 10s")

		// Calculate metrics
		successRate := float64(successfulClients) / float64(clientCount) * 100
		t.Logf("Client success rate: %.2f%%", successRate)
		t.Logf("Total messages sent: %d", totalMessagesSent)
		t.Logf("Duration: %v", duration)
	})

	t.Run("ResourceIsolation", func(t *testing.T) {
		// Create clients with different behaviors
		goodClient, err := grpc.Dial(addr, grpc.WithInsecure())
		require.NoError(t, err)
		defer goodClient.Close()

		badClient, err := grpc.Dial(addr, grpc.WithInsecure())
		require.NoError(t, err)
		defer badClient.Close()

		// Good client sends normal messages
		goodStream, err := pb.NewChatServiceClient(goodClient).Chat(ctx)
		require.NoError(t, err)

		// Bad client floods with messages
		badStream, err := pb.NewChatServiceClient(badClient).Chat(ctx)
		require.NoError(t, err)

		// Start good client messaging
		goodMessages := 0
		go func() {
			for i := 0; i < 10; i++ {
				err := goodStream.Send(&pb.ChatRequest{
					Request: &pb.ChatRequest_Message{
						Message: &pb.ChatMessage{
							SessionId: "good-client",
							SenderId:  "user",
							Content:   "Normal message",
							Timestamp: time.Now().Unix(),
						},
					},
				})
				if err == nil {
					goodMessages++
				}
				time.Sleep(100 * time.Millisecond)
			}
		}()

		// Bad client floods
		go func() {
			for i := 0; i < 1000; i++ {
				badStream.Send(&pb.ChatRequest{
					Request: &pb.ChatRequest_Message{
						Message: &pb.ChatMessage{
							SessionId: "bad-client",
							SenderId:  "user",
							Content:   strings.Repeat("spam", 1000),
							Timestamp: time.Now().Unix(),
						},
					},
				})
			}
		}()

		// Wait for good client to finish
		time.Sleep(1200 * time.Millisecond)

		// Good client should still succeed despite bad client
		assert.Greater(t, goodMessages, 8, "Good client should send most messages despite flood")
	})

	t.Run("GracefulDegradation", func(t *testing.T) {
		// Track server metrics
		var activeStreams int32
		var rejectedConnections int32

		// Create many clients to saturate server
		for i := 0; i < 200; i++ {
			go func(clientID int) {
				conn, err := grpc.Dial(addr, grpc.WithInsecure())
				if err != nil {
					atomic.AddInt32(&rejectedConnections, 1)
					return
				}
				defer conn.Close()

				client := pb.NewChatServiceClient(conn)
				stream, err := client.Chat(ctx)
				if err != nil {
					if status.Code(err) == codes.ResourceExhausted {
						atomic.AddInt32(&rejectedConnections, 1)
					}
					return
				}

				atomic.AddInt32(&activeStreams, 1)
				defer atomic.AddInt32(&activeStreams, -1)

				// Keep stream open
				time.Sleep(500 * time.Millisecond)
				stream.CloseSend()
			}(i)
		}

		// Wait for all clients to connect/fail
		time.Sleep(1 * time.Second)

		// Server should accept some but gracefully reject others
		assert.Greater(t, activeStreams, int32(50), "Should handle many concurrent streams")
		assert.Greater(t, rejectedConnections, int32(0), "Should reject excess connections gracefully")

		t.Logf("Active streams: %d, Rejected: %d", activeStreams, rejectedConnections)
	})
}

// Helper functions

func setupTestGRPCServer(t *testing.T) (*grpc.Server, string) {
	lis, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)

	server := grpc.NewServer()
	chatService := createTestGRPCServer(t)
	pb.RegisterChatServiceServer(server, chatService)

	go server.Serve(lis)

	return server, lis.Addr().String()
}

func createTestGRPCServer(t *testing.T) *grpcserver.ChatService {
	// Setup test project and registry
	_, cleanup := testutil.SetupTestProject(t)
	t.Cleanup(cleanup)

	reg := registry.NewComponentRegistry()
	err := reg.Initialize(context.Background(), registry.Config{})
	require.NoError(t, err)

	// Create a simple in-memory event bus for testing
	eventBus := &inMemoryEventBus{
		handlers: make(map[string][]func(event interface{})),
	}

	// Create chat service
	chatService := grpcserver.NewChatService(reg, eventBus)

	return chatService
}

// controlledListener wraps a listener to simulate network issues
type controlledListener struct {
	net.Listener
	mu     sync.RWMutex
	active bool
}

func (cl *controlledListener) Accept() (net.Conn, error) {
	cl.mu.RLock()
	active := cl.active
	cl.mu.RUnlock()

	if !active {
		time.Sleep(100 * time.Millisecond)
		return nil, fmt.Errorf("network unavailable")
	}

	return cl.Listener.Accept()
}

func (cl *controlledListener) SetActive(active bool) {
	cl.mu.Lock()
	cl.active = active
	cl.mu.Unlock()
}
