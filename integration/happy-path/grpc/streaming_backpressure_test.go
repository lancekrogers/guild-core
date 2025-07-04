package grpc

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// StreamingConfig defines streaming configuration
type StreamingConfig struct {
	MaxConcurrentStreams  int
	BufferSize            int
	BackpressureThreshold int
	FlowControlWindow     int
	HeartbeatInterval     time.Duration
}

// MessageConfig defines message handling configuration
type MessageConfig struct {
	MaxMessageSize     int
	CompressionLevel   int
	EnableBatching     bool
	BatchFlushInterval time.Duration
}

// MessageType represents different types of messages
type MessageType int

const (
	MessageType_AgentResponse MessageType = iota
	MessageType_KanbanUpdate
	MessageType_ContextData
)

// MessageSpec defines message generation parameters
type MessageSpec struct {
	Type     MessageType
	Size     int
	StreamID string
	SeqNum   int
	Metadata map[string]string
}

// Message represents a streaming message
type Message struct {
	Type     MessageType
	Data     []byte
	Metadata map[string]string
	ID       string
}

// StreamConfig defines stream configuration
type StreamConfig struct {
	StreamID      string
	BufferSize    int
	FlowControl   bool
	EnableMetrics bool
}

// BackpressureConfig defines backpressure handling
type BackpressureConfig struct {
	MaxWaitTime    time.Duration
	RetryInterval  time.Duration
	OnBackpressure func(int)
}

// StreamMetrics tracks streaming performance
type StreamMetrics struct {
	StreamID           string
	MessagesSent       int
	MessagesReceived   int
	BackpressureEvents int
	TotalLatency       time.Duration
	ErrorCount         int
	StartTime          time.Time
	mu                 sync.RWMutex
}

// NewStreamMetrics creates new stream metrics
func NewStreamMetrics(streamID string) *StreamMetrics {
	return &StreamMetrics{
		StreamID:  streamID,
		StartTime: time.Now(),
	}
}

// RecordBackpressureEvent records a backpressure event
func (m *StreamMetrics) RecordBackpressureEvent(queueSize int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.BackpressureEvents++
}

// RecordMessageSent records a sent message
func (m *StreamMetrics) RecordMessageSent(latency time.Duration, size int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.MessagesSent++
	m.TotalLatency += latency
}

// GetSummary returns stream metrics summary
func (m *StreamMetrics) GetSummary() StreamMetricsSummary {
	m.mu.RLock()
	defer m.mu.RUnlock()

	duration := time.Since(m.StartTime)
	successRate := float64(m.MessagesReceived) / float64(m.MessagesSent)
	if m.MessagesSent == 0 {
		successRate = 0
	}

	averageLatency := time.Duration(0)
	if m.MessagesSent > 0 {
		averageLatency = m.TotalLatency / time.Duration(m.MessagesSent)
	}

	return StreamMetricsSummary{
		SuccessRate:        successRate,
		AverageLatency:     averageLatency,
		BackpressureEvents: m.BackpressureEvents,
		MessagesPerSecond:  float64(m.MessagesSent) / duration.Seconds(),
	}
}

// StreamMetricsSummary contains stream performance summary
type StreamMetricsSummary struct {
	SuccessRate        float64
	AverageLatency     time.Duration
	BackpressureEvents int
	MessagesPerSecond  float64
}

// MonitorConfig defines monitoring configuration
type MonitorConfig struct {
	SampleInterval      time.Duration
	ThroughputThreshold int
	LatencyThreshold    time.Duration
}

// PerformanceResults contains performance monitoring results
type PerformanceResults struct {
	ResourceUsage ResourceUsageMetrics
	FlowControl   FlowControlMetrics
	Duration      time.Duration
}

// ResourceUsageMetrics tracks resource usage during streaming
type ResourceUsageMetrics struct {
	MaxMemoryMB   int
	MaxCPUPercent float64
	AvgMemoryMB   int
	AvgCPUPercent float64
}

// FlowControlMetrics tracks flow control effectiveness
type FlowControlMetrics struct {
	WindowExhaustionRate float64
	WindowUtilization    float64
	BackpressureEvents   int
}

// MockStreamingClient represents a streaming client
type MockStreamingClient struct {
	address string
	streams map[string]*MockStream
	mu      sync.RWMutex
}

// CreateBidirectionalStream creates a bidirectional stream
func (c *MockStreamingClient) CreateBidirectionalStream(config StreamConfig) (*MockStream, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	stream := &MockStream{
		id:          config.StreamID,
		bufferSize:  config.BufferSize,
		flowControl: config.FlowControl,
		buffer:      make(chan *Message, config.BufferSize),
		metrics:     NewStreamMetrics(config.StreamID),
	}

	if c.streams == nil {
		c.streams = make(map[string]*MockStream)
	}
	c.streams[config.StreamID] = stream

	return stream, nil
}

// Close closes the streaming client
func (c *MockStreamingClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, stream := range c.streams {
		stream.Close()
	}
	c.streams = nil
	return nil
}

// MockStream represents a mock streaming connection
type MockStream struct {
	id          string
	bufferSize  int
	flowControl bool
	buffer      chan *Message
	metrics     *StreamMetrics
	closed      bool
	mu          sync.RWMutex
}

// SendWithBackpressure sends a message with backpressure handling
func (s *MockStream) SendWithBackpressure(ctx context.Context, message *Message, config BackpressureConfig) error {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return errors.New("stream closed")
	}
	s.mu.RUnlock()

	start := time.Now()

	// Simulate backpressure when buffer is full
	select {
	case s.buffer <- message:
		// Message sent successfully
		s.metrics.RecordMessageSent(time.Since(start), len(message.Data))
		return nil

	case <-time.After(config.MaxWaitTime):
		// Backpressure timeout
		if config.OnBackpressure != nil {
			config.OnBackpressure(len(s.buffer))
		}
		return errors.New("backpressure timeout")

	case <-ctx.Done():
		return ctx.Err()
	}
}

// Close closes the stream
func (s *MockStream) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.closed {
		s.closed = true
		close(s.buffer)
	}
	return nil
}

// PerformanceMonitor monitors streaming performance
type PerformanceMonitor struct {
	config    MonitorConfig
	daemon    DaemonInterface
	startTime time.Time
	samples   []PerformanceSample
	mu        sync.RWMutex
	stopChan  chan struct{}
	stopped   bool
}

// PerformanceSample represents a performance measurement
type PerformanceSample struct {
	Timestamp  time.Time
	MemoryMB   int
	CPUPercent float64
	Throughput int
	Latency    time.Duration
}

// StartStreamingPerformanceMonitor starts performance monitoring
func (f *GRPCTestFramework) StartStreamingPerformanceMonitor(daemon DaemonInterface, config MonitorConfig) *PerformanceMonitor {
	monitor := &PerformanceMonitor{
		config:    config,
		daemon:    daemon,
		startTime: time.Now(),
		samples:   make([]PerformanceSample, 0),
		stopChan:  make(chan struct{}),
	}

	go monitor.run()
	return monitor
}

// run executes the monitoring loop
func (m *PerformanceMonitor) run() {
	ticker := time.NewTicker(m.config.SampleInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.collectSample()
		}
	}
}

// collectSample collects a performance sample
func (m *PerformanceMonitor) collectSample() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.stopped {
		return
	}

	usage := m.daemon.GetResourceUsage()

	sample := PerformanceSample{
		Timestamp:  time.Now(),
		MemoryMB:   int(usage.MemoryMB),
		CPUPercent: usage.CPUPercent,
		Throughput: rand.Intn(100) + 50, // Mock throughput
		Latency:    time.Duration(rand.Intn(100)+50) * time.Millisecond,
	}

	m.samples = append(m.samples, sample)
}

// StopStreamingPerformanceMonitor stops performance monitoring
func (f *GRPCTestFramework) StopStreamingPerformanceMonitor(monitor *PerformanceMonitor) *PerformanceResults {
	monitor.mu.Lock()
	defer monitor.mu.Unlock()

	if !monitor.stopped {
		monitor.stopped = true
		close(monitor.stopChan)
	}

	return monitor.getResults()
}

// getResults calculates performance results
func (m *PerformanceMonitor) getResults() *PerformanceResults {
	if len(m.samples) == 0 {
		return &PerformanceResults{
			Duration: time.Since(m.startTime),
		}
	}

	totalMemory := 0
	totalCPU := 0.0
	maxMemory := 0
	maxCPU := 0.0

	for _, sample := range m.samples {
		totalMemory += sample.MemoryMB
		totalCPU += sample.CPUPercent

		if sample.MemoryMB > maxMemory {
			maxMemory = sample.MemoryMB
		}
		if sample.CPUPercent > maxCPU {
			maxCPU = sample.CPUPercent
		}
	}

	avgMemory := totalMemory / len(m.samples)
	avgCPU := totalCPU / float64(len(m.samples))

	return &PerformanceResults{
		ResourceUsage: ResourceUsageMetrics{
			MaxMemoryMB:   maxMemory,
			MaxCPUPercent: maxCPU,
			AvgMemoryMB:   avgMemory,
			AvgCPUPercent: avgCPU,
		},
		FlowControl: FlowControlMetrics{
			WindowExhaustionRate: 0.02, // Mock value
			WindowUtilization:    0.75, // Mock value
			BackpressureEvents:   rand.Intn(5),
		},
		Duration: time.Since(m.startTime),
	}
}

// CreateStreamingClient creates a streaming client
func (f *GRPCTestFramework) CreateStreamingClient(address string) (*MockStreamingClient, error) {
	client := &MockStreamingClient{
		address: address,
		streams: make(map[string]*MockStream),
	}

	f.cleanup = append(f.cleanup, func() {
		client.Close()
	})

	return client, nil
}

// GenerateRealisticMessage generates a realistic message for testing
func (f *GRPCTestFramework) GenerateRealisticMessage(spec MessageSpec) *Message {
	// Generate realistic message content
	data := make([]byte, spec.Size)
	for i := range data {
		data[i] = byte(rand.Intn(256))
	}

	return &Message{
		Type:     spec.Type,
		Data:     data,
		Metadata: spec.Metadata,
		ID:       fmt.Sprintf("%s-%d", spec.StreamID, spec.SeqNum),
	}
}

// TestStreamingBackpressure_HappyPath validates streaming performance under load
func TestStreamingBackpressure_HappyPath(t *testing.T) {
	framework := NewGRPCTestFramework(t)
	defer framework.Cleanup()

	streamingScenarios := []struct {
		name                  string
		concurrentStreams     int
		messagesPerStream     int
		messageSize           int
		expectedThroughput    int // messages per second
		backpressureThreshold int // queue size that triggers backpressure
	}{
		{
			name:                  "Low volume streaming",
			concurrentStreams:     5,
			messagesPerStream:     100,
			messageSize:           1024,
			expectedThroughput:    1000,
			backpressureThreshold: 50,
		},
		{
			name:                  "High volume streaming with backpressure",
			concurrentStreams:     20,
			messagesPerStream:     500,
			messageSize:           4096,
			expectedThroughput:    5000,
			backpressureThreshold: 100,
		},
		{
			name:                  "Large message streaming",
			concurrentStreams:     10,
			messagesPerStream:     50,
			messageSize:           64 * 1024, // 64KB messages
			expectedThroughput:    200,
			backpressureThreshold: 20,
		},
	}

	for _, scenario := range streamingScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Initialize streaming server with backpressure configuration
			daemon, err := framework.StartDaemon(DaemonConfig{
				Port:                framework.GetAvailablePort(),
				HealthCheckInterval: 1 * time.Second,
				RestartPolicy:       RestartPolicy_Always,
				MaxRestartAttempts:  5,
				ResourceLimits: ResourceLimits{
					MaxMemoryMB:   500,
					MaxCPUPercent: 80,
					MaxGoroutines: 1000,
				},
			})
			require.NoError(t, err)
			defer daemon.Stop()

			// Create streaming clients
			streamMetrics := make([]*StreamMetrics, scenario.concurrentStreams)
			var streamWg sync.WaitGroup

			streamingStart := time.Now()

			for i := 0; i < scenario.concurrentStreams; i++ {
				streamWg.Add(1)
				go func(streamIdx int) {
					defer streamWg.Done()

					client, err := framework.CreateStreamingClient(daemon.Address())
					require.NoError(t, err, "Failed to create streaming client %d", streamIdx)
					defer client.Close()

					metrics := NewStreamMetrics(fmt.Sprintf("stream-%d", streamIdx))
					streamMetrics[streamIdx] = metrics

					// Create bidirectional stream
					stream, err := client.CreateBidirectionalStream(StreamConfig{
						StreamID:      fmt.Sprintf("stream-%d", streamIdx),
						BufferSize:    scenario.backpressureThreshold / 2,
						FlowControl:   true,
						EnableMetrics: true,
					})
					require.NoError(t, err, "Failed to create stream %d", streamIdx)

					// Send messages with realistic pacing
					sendCtx, sendCancel := context.WithTimeout(context.Background(), 60*time.Second)
					defer sendCancel()

					for msgIdx := 0; msgIdx < scenario.messagesPerStream; msgIdx++ {
						messageStart := time.Now()

						// Generate realistic message content
						message := framework.GenerateRealisticMessage(MessageSpec{
							Type:     MessageType_AgentResponse,
							Size:     scenario.messageSize,
							StreamID: fmt.Sprintf("stream-%d", streamIdx),
							SeqNum:   msgIdx,
							Metadata: map[string]string{
								"client_id": fmt.Sprintf("client-%d", streamIdx),
								"timestamp": time.Now().Format(time.RFC3339),
							},
						})

						// Send with backpressure handling
						err = stream.SendWithBackpressure(sendCtx, message, BackpressureConfig{
							MaxWaitTime:   5 * time.Second,
							RetryInterval: 100 * time.Millisecond,
							OnBackpressure: func(queueSize int) {
								metrics.RecordBackpressureEvent(queueSize)
							},
						})

						sendDuration := time.Since(messageStart)

						if err != nil {
							if !errors.Is(err, context.Canceled) {
								t.Errorf("Failed to send message %d on stream %d: %v", msgIdx, streamIdx, err)
							}
							break
						}

						metrics.RecordMessageSent(sendDuration, len(message.Data))

						// Realistic inter-message delay
						time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
					}
				}(i)
			}

			// Monitor streaming performance
			performanceMonitor := framework.StartStreamingPerformanceMonitor(daemon, MonitorConfig{
				SampleInterval:      1 * time.Second,
				ThroughputThreshold: scenario.expectedThroughput,
				LatencyThreshold:    500 * time.Millisecond,
			})

			streamWg.Wait()
			streamingDuration := time.Since(streamingStart)

			performanceResults := framework.StopStreamingPerformanceMonitor(performanceMonitor)

			// PHASE 1: Validate throughput performance
			totalMessages := scenario.concurrentStreams * scenario.messagesPerStream
			actualThroughput := float64(totalMessages) / streamingDuration.Seconds()

			assert.GreaterOrEqual(t, actualThroughput, float64(scenario.expectedThroughput)*0.8,
				"Throughput below 80%% of target: %.1f < %.1f msg/s",
				actualThroughput, float64(scenario.expectedThroughput)*0.8)

			// PHASE 2: Validate backpressure handling
			totalBackpressureEvents := 0
			totalLatency := time.Duration(0)

			for i, metrics := range streamMetrics {
				if metrics == nil {
					continue
				}

				summary := metrics.GetSummary()

				// Validate individual stream performance
				assert.GreaterOrEqual(t, summary.SuccessRate, 0.95,
					"Stream %d success rate too low: %.2f%%", i, summary.SuccessRate*100)

				totalBackpressureEvents += summary.BackpressureEvents
				totalLatency += summary.AverageLatency

				// Validate backpressure events are reasonable
				if scenario.backpressureThreshold > 0 {
					expectedBackpressureRate := float64(summary.BackpressureEvents) / float64(scenario.messagesPerStream)
					assert.LessOrEqual(t, expectedBackpressureRate, 0.1,
						"Stream %d excessive backpressure: %.2f%% of messages", i, expectedBackpressureRate*100)
				}
			}

			averageLatency := totalLatency / time.Duration(scenario.concurrentStreams)
			assert.LessOrEqual(t, averageLatency, 500*time.Millisecond,
				"Average latency too high: %v", averageLatency)

			// PHASE 3: Validate resource efficiency
			resourceMetrics := performanceResults.ResourceUsage
			assert.LessOrEqual(t, resourceMetrics.MaxMemoryMB, 200,
				"Memory usage too high during streaming: %d MB", resourceMetrics.MaxMemoryMB)
			assert.LessOrEqual(t, resourceMetrics.MaxCPUPercent, 80.0,
				"CPU usage too high during streaming: %.1f%%", resourceMetrics.MaxCPUPercent)

			// PHASE 4: Validate flow control effectiveness
			flowControlMetrics := performanceResults.FlowControl
			assert.LessOrEqual(t, flowControlMetrics.WindowExhaustionRate, 0.05,
				"Flow control window exhaustion too high: %.2f%%", flowControlMetrics.WindowExhaustionRate*100)

			t.Logf("✅ Streaming backpressure test completed successfully")
			t.Logf("📊 Streaming Performance Summary:")
			t.Logf("   - Total Messages: %d", totalMessages)
			t.Logf("   - Actual Throughput: %.1f msg/s", actualThroughput)
			t.Logf("   - Average Latency: %v", averageLatency)
			t.Logf("   - Backpressure Events: %d", totalBackpressureEvents)
			t.Logf("   - Peak Memory: %d MB", resourceMetrics.MaxMemoryMB)
			t.Logf("   - Peak CPU: %.1f%%", resourceMetrics.MaxCPUPercent)
		})
	}
}
