package agent

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/guild-ventures/guild-core/pkg/providers/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock provider for testing
type mockAIProvider struct {
	name         string
	capabilities []string
	shouldError  bool
	response     string
}

func (m *mockAIProvider) ChatCompletion(ctx context.Context, req interfaces.ChatRequest) (*interfaces.ChatResponse, error) {
	if m.shouldError {
		return nil, fmt.Errorf("provider error")
	}
	return &interfaces.ChatResponse{
		ID:    "test-response",
		Model: "mock-model",
		Choices: []interfaces.ChatChoice{
			{
				Index: 0,
				Message: interfaces.ChatMessage{
					Role:    "assistant",
					Content: m.response,
				},
				FinishReason: "stop",
			},
		},
		Usage: interfaces.UsageInfo{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}, nil
}

func (m *mockAIProvider) StreamChatCompletion(ctx context.Context, req interfaces.ChatRequest) (interfaces.ChatStream, error) {
	return nil, fmt.Errorf("streaming not supported")
}

func (m *mockAIProvider) CreateEmbedding(ctx context.Context, req interfaces.EmbeddingRequest) (*interfaces.EmbeddingResponse, error) {
	return nil, fmt.Errorf("embeddings not supported")
}

func (m *mockAIProvider) GetCapabilities() interfaces.ProviderCapabilities {
	return interfaces.ProviderCapabilities{
		MaxTokens:          4096,
		ContextWindow:      8192,
		SupportsVision:     false,
		SupportsTools:      false,
		SupportsStream:     false,
		SupportsEmbeddings: false,
		Models: []interfaces.ModelInfo{
			{
				ID:            "mock-model",
				Name:          "Mock Model",
				ContextWindow: 8192,
				MaxOutput:     4096,
			},
		},
	}
}

// Test newContextAwareAgent
func TestNewContextAwareAgent(t *testing.T) {
	tests := []struct {
		name          string
		id            string
		agentName     string
		providers     map[string]interfaces.Provider
		capabilities  []string
		expectError   bool
		validateAgent func(t *testing.T, agent *ContextAwareAgent)
	}{
		{
			name:      "successful creation with providers",
			id:        "test-agent-1",
			agentName: "Test Agent 1",
			providers: map[string]interfaces.Provider{
				"provider1": &mockProvider{name: "provider1", capabilities: []string{"code", "analysis"}},
				"provider2": &mockProvider{name: "provider2", capabilities: []string{"translation"}},
			},
			capabilities: []string{"code", "analysis", "translation"},
			expectError:  false,
			validateAgent: func(t *testing.T, agent *ContextAwareAgent) {
				assert.Equal(t, "test-agent-1", agent.ID)
				assert.Equal(t, "Test Agent 1", agent.Name)
				assert.Equal(t, 3, len(agent.Capabilities))
				assert.Equal(t, 2, len(agent.providers))
			},
		},
		{
			name:         "creation with nil providers",
			id:           "test-agent-2",
			agentName:    "Test Agent 2",
			providers:    nil,
			capabilities: []string{"general"},
			expectError:  false,
			validateAgent: func(t *testing.T, agent *ContextAwareAgent) {
				assert.Equal(t, 0, len(agent.providers))
			},
		},
		{
			name:         "creation with empty capabilities",
			id:           "test-agent-3",
			agentName:    "Test Agent 3",
			providers:    map[string]interfaces.Provider{},
			capabilities: []string{},
			expectError:  false,
			validateAgent: func(t *testing.T, agent *ContextAwareAgent) {
				assert.Equal(t, 0, len(agent.Capabilities))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := newContextAwareAgent(tt.id, tt.agentName, tt.providers, tt.capabilities)
			require.NotNil(t, agent)
			
			if tt.validateAgent != nil {
				tt.validateAgent(t, agent)
			}
		})
	}
}

// Test Execute method
func TestContextAwareAgent_Execute(t *testing.T) {
	tests := []struct {
		name        string
		setupAgent  func() *ContextAwareAgent
		request     string
		expectError bool
		expectMsg   string
	}{
		{
			name: "successful execution with default provider",
			setupAgent: func() *ContextAwareAgent {
				providers := map[string]interfaces.Provider{
					"default": &mockProvider{name: "default", capabilities: []string{"general"}, response: "Success response"},
				}
				agent := newContextAwareAgent("test-1", "Test Agent", providers, []string{"general"})
				agent.defaultProvider = "default"
				return agent
			},
			request:     "test request",
			expectError: false,
			expectMsg:   "Success response",
		},
		{
			name: "execution with no providers",
			setupAgent: func() *ContextAwareAgent {
				return newContextAwareAgent("test-2", "Test Agent", nil, []string{"general"})
			},
			request:     "test request",
			expectError: true,
			expectMsg:   "no providers available",
		},
		{
			name: "execution with provider error",
			setupAgent: func() *ContextAwareAgent {
				providers := map[string]interfaces.Provider{
					"error-provider": &mockProvider{name: "error", shouldError: true},
				}
				agent := newContextAwareAgent("test-3", "Test Agent", providers, []string{"general"})
				agent.defaultProvider = "error-provider"
				return agent
			},
			request:     "test request",
			expectError: true,
			expectMsg:   "provider error",
		},
		{
			name: "execution with code analysis request",
			setupAgent: func() *ContextAwareAgent {
				providers := map[string]interfaces.Provider{
					"code-provider": &mockProvider{
						name:         "code",
						capabilities: []string{"code", "analysis"},
						response:     "Code analysis complete",
					},
				}
				return newContextAwareAgent("test-4", "Code Agent", providers, []string{"code", "analysis"})
			},
			request:     "analyze this code: func main() {}",
			expectError: false,
			expectMsg:   "Code analysis complete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			agent := tt.setupAgent()
			
			response, err := agent.Execute(ctx, tt.request)
			
			if tt.expectError {
				assert.Error(t, err)
				if tt.expectMsg != "" {
					assert.Contains(t, err.Error(), tt.expectMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectMsg, response)
			}
		})
	}
}

// Test executeWithContext
func TestContextAwareAgent_executeWithContext(t *testing.T) {
	ctx := context.Background()
	
	tests := []struct {
		name        string
		setupAgent  func() *ContextAwareAgent
		request     string
		expectError bool
	}{
		{
			name: "successful execution",
			setupAgent: func() *ContextAwareAgent {
				providers := map[string]interfaces.Provider{
					"test": &mockProvider{response: "executed successfully"},
				}
				agent := newContextAwareAgent("test-1", "Test", providers, []string{"general"})
				agent.defaultProvider = "test"
				return agent
			},
			request:     "test request",
			expectError: false,
		},
		{
			name: "execution with timeout",
			setupAgent: func() *ContextAwareAgent {
				providers := map[string]interfaces.Provider{
					"slow": &mockProvider{
						response: "slow response",
						shouldError: false,
					},
				}
				agent := newContextAwareAgent("test-2", "Test", providers, []string{"general"})
				agent.defaultProvider = "slow"
				return agent
			},
			request:     "test request",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := tt.setupAgent()
			
			result, err := agent.executeWithContext(ctx, tt.request)
			
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, result)
			}
		})
	}
}

// Test selectProvider
func TestContextAwareAgent_selectProvider(t *testing.T) {
	tests := []struct {
		name           string
		setupAgent     func() *ContextAwareAgent
		request        string
		expectedName   string
		expectProvider bool
	}{
		{
			name: "select code provider for code request",
			setupAgent: func() *ContextAwareAgent {
				providers := map[string]interfaces.Provider{
					"general": &mockProvider{name: "general", capabilities: []string{"general"}},
					"code":    &mockProvider{name: "code", capabilities: []string{"code", "analysis"}},
				}
				return newContextAwareAgent("test-1", "Test", providers, []string{"general", "code"})
			},
			request:        "analyze this function",
			expectedName:   "code",
			expectProvider: true,
		},
		{
			name: "use default provider when set",
			setupAgent: func() *ContextAwareAgent {
				providers := map[string]interfaces.Provider{
					"provider1": &mockProvider{name: "provider1"},
					"provider2": &mockProvider{name: "provider2"},
				}
				agent := newContextAwareAgent("test-2", "Test", providers, []string{"general"})
				agent.defaultProvider = "provider2"
				return agent
			},
			request:        "any request",
			expectedName:   "provider2",
			expectProvider: true,
		},
		{
			name: "no provider available",
			setupAgent: func() *ContextAwareAgent {
				return newContextAwareAgent("test-3", "Test", nil, []string{"general"})
			},
			request:        "test request",
			expectProvider: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := tt.setupAgent()
			
			provider, name := agent.selectProvider(tt.request)
			
			if tt.expectProvider {
				assert.NotNil(t, provider)
				assert.Equal(t, tt.expectedName, name)
			} else {
				assert.Nil(t, provider)
				assert.Empty(t, name)
			}
		})
	}
}

// Test determineTaskType
func TestContextAwareAgent_determineTaskType(t *testing.T) {
	agent := &ContextAwareAgent{}
	
	tests := []struct {
		name     string
		request  string
		expected string
	}{
		{"code analysis", "analyze this code", "code_analysis"},
		{"code generation", "generate a function", "code_generation"},
		{"debugging", "debug this error", "debugging"},
		{"translation", "translate to Spanish", "translation"},
		{"research", "research this topic", "research"},
		{"documentation", "write documentation", "documentation"},
		{"general task", "what is the weather", "general"},
		{"mixed keywords", "analyze and translate this code", "code_analysis"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := agent.determineTaskType(tt.request)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test createSystemPrompt
func TestContextAwareAgent_createSystemPrompt(t *testing.T) {
	tests := []struct {
		name     string
		agent    *ContextAwareAgent
		taskType string
		expected string
	}{
		{
			name: "code analysis prompt",
			agent: &ContextAwareAgent{
				Name:         "Code Agent",
				Capabilities: []string{"code", "analysis"},
				systemPrompt: "Base prompt",
			},
			taskType: "code_analysis",
			expected: "code analysis",
		},
		{
			name: "general task with custom prompt",
			agent: &ContextAwareAgent{
				Name:         "General Agent",
				Capabilities: []string{"general"},
				systemPrompt: "Custom system prompt",
			},
			taskType: "general",
			expected: "Custom system prompt",
		},
		{
			name: "documentation task",
			agent: &ContextAwareAgent{
				Name:         "Doc Agent",
				Capabilities: []string{"documentation"},
			},
			taskType: "documentation",
			expected: "documentation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.agent.createSystemPrompt(tt.taskType)
			assert.Contains(t, strings.ToLower(result), tt.expected)
		})
	}
}

// Test postProcessResult
func TestContextAwareAgent_postProcessResult(t *testing.T) {
	agent := &ContextAwareAgent{}
	
	tests := []struct {
		name     string
		result   string
		taskType string
		expected string
	}{
		{
			name:     "code analysis result",
			result:   "Function analysis complete",
			taskType: "code_analysis",
			expected: "Function analysis complete",
		},
		{
			name:     "empty result handling",
			result:   "",
			taskType: "general",
			expected: "No response generated",
		},
		{
			name:     "whitespace trimming",
			result:   "  Result with spaces  \n",
			taskType: "general",
			expected: "Result with spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := agent.postProcessResult(tt.result, tt.taskType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test getter methods
func TestContextAwareAgent_Getters(t *testing.T) {
	agent := &ContextAwareAgent{
		ID:           "test-id",
		Name:         "Test Agent",
		Capabilities: []string{"cap1", "cap2"},
		Status:       "active",
		AgentType:    "context",
		metadata: map[string]interface{}{
			"key1": "value1",
			"key2": 42,
		},
	}

	// Test GetID
	assert.Equal(t, "test-id", agent.GetID())

	// Test GetName
	assert.Equal(t, "Test Agent", agent.GetName())

	// Test GetCapabilities
	caps := agent.GetCapabilities()
	assert.Equal(t, 2, len(caps))
	assert.Contains(t, caps, "cap1")
	assert.Contains(t, caps, "cap2")

	// Test GetStatus
	assert.Equal(t, "active", agent.GetStatus())

	// Test GetAgentType
	assert.Equal(t, "context", agent.GetAgentType())

	// Test GetMetadata
	meta := agent.GetMetadata()
	assert.Equal(t, "value1", meta["key1"])
	assert.Equal(t, 42, meta["key2"])
}

// Test setter methods
func TestContextAwareAgent_Setters(t *testing.T) {
	agent := &ContextAwareAgent{
		providers: map[string]interfaces.Provider{
			"provider1": &mockProvider{},
		},
		metadata: make(map[string]interface{}),
	}

	// Test SetDefaultProvider
	agent.SetDefaultProvider("provider1")
	assert.Equal(t, "provider1", agent.defaultProvider)

	// Test SetSystemPrompt
	agent.SetSystemPrompt("New system prompt")
	assert.Equal(t, "New system prompt", agent.systemPrompt)

	// Test UpdateCapabilities
	agent.UpdateCapabilities([]string{"new1", "new2"})
	assert.Equal(t, 2, len(agent.Capabilities))
	assert.Contains(t, agent.Capabilities, "new1")
	assert.Contains(t, agent.Capabilities, "new2")

	// Test AddMetadata
	agent.AddMetadata("testKey", "testValue")
	assert.Equal(t, "testValue", agent.metadata["testKey"])
}

// Test Reset method
func TestContextAwareAgent_Reset(t *testing.T) {
	agent := &ContextAwareAgent{
		Status: "busy",
		metadata: map[string]interface{}{
			"key1":      "value1",
			"key2":      "value2",
			"timestamp": time.Now(),
		},
	}

	// Add some metadata
	agent.AddMetadata("tempKey", "tempValue")
	assert.Equal(t, 4, len(agent.metadata))

	// Reset
	agent.Reset()

	// Verify reset
	assert.Equal(t, "ready", agent.Status)
	assert.Equal(t, 0, len(agent.metadata))
}

// Test helper functions
func TestContextAwareAgent_HelperFunctions(t *testing.T) {
	agent := &ContextAwareAgent{}

	// Test containsAny
	t.Run("containsAny", func(t *testing.T) {
		assert.True(t, agent.containsAny("analyze this code", []string{"analyze", "review"}))
		assert.True(t, agent.containsAny("review the function", []string{"analyze", "review"}))
		assert.False(t, agent.containsAny("write documentation", []string{"analyze", "review"}))
		assert.True(t, agent.containsAny("ANALYZE THIS", []string{"analyze"})) // case insensitive
	})

	// Test contains
	t.Run("contains", func(t *testing.T) {
		assert.True(t, agent.contains("hello world", "world"))
		assert.True(t, agent.contains("HELLO WORLD", "world")) // case insensitive
		assert.False(t, agent.contains("hello world", "xyz"))
	})

	// Test findSubstring
	t.Run("findSubstring", func(t *testing.T) {
		found, index := agent.findSubstring("hello world", "world")
		assert.True(t, found)
		assert.Equal(t, 6, index)

		found, index = agent.findSubstring("HELLO WORLD", "world") // case insensitive
		assert.True(t, found)
		assert.Equal(t, 6, index)

		found, index = agent.findSubstring("hello world", "xyz")
		assert.False(t, found)
		assert.Equal(t, -1, index)
	})

	// Test toLowerCase
	t.Run("toLowerCase", func(t *testing.T) {
		assert.Equal(t, "hello world", agent.toLowerCase("HELLO WORLD"))
		assert.Equal(t, "mixed case", agent.toLowerCase("MiXeD CaSe"))
		assert.Equal(t, "123 test", agent.toLowerCase("123 TEST"))
	})
}

// Test concurrent execution
func TestContextAwareAgent_ConcurrentExecution(t *testing.T) {
	providers := map[string]interfaces.Provider{
		"concurrent": &mockProvider{
			name:     "concurrent",
			response: "concurrent response",
		},
	}
	agent := newContextAwareAgent("concurrent-test", "Concurrent Agent", providers, []string{"general"})
	agent.defaultProvider = "concurrent"

	// Run multiple concurrent executions
	ctx := context.Background()
	numGoroutines := 10
	results := make(chan string, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			response, err := agent.Execute(ctx, fmt.Sprintf("request %d", id))
			if err != nil {
				errors <- err
			} else {
				results <- response
			}
		}(i)
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		select {
		case err := <-errors:
			t.Errorf("Concurrent execution failed: %v", err)
		case result := <-results:
			assert.Equal(t, "concurrent response", result)
		case <-time.After(5 * time.Second):
			t.Error("Timeout waiting for concurrent execution")
		}
	}
}