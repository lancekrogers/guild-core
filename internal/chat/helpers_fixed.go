package chat

import (
	"fmt"
	"time"
)

// Tool execution message handlers (FIXED VERSION)
func (m *ChatModel) handleToolExecutionComplete(msg toolExecutionCompleteMsg) {
	// Get tool from active tools
	tool, exists := m.activeTools[msg.executionID]
	if exists {
		// Remove from active tools
		delete(m.activeTools, msg.executionID)

		// Add completion message
		m.messages = append(m.messages, Message{
			Type:      msgToolComplete,
			Content:   fmt.Sprintf("✅ %s: Completed", msg.executionID),
			AgentID:   tool.AgentID,
			Timestamp: time.Now(),
			Metadata: map[string]string{
				"execution_id": msg.executionID,
				"result":       msg.result,
			},
		})
	} else {
		// Add message without agent info
		m.messages = append(m.messages, Message{
			Type:      msgToolComplete,
			Content:   fmt.Sprintf("✅ Completed execution %s", msg.executionID),
			Timestamp: time.Now(),
			Metadata: map[string]string{
				"execution_id": msg.executionID,
				"result":       msg.result,
			},
		})
	}
	m.updateMessagesView()
}

func (m *ChatModel) handleToolExecutionError(msg toolExecutionErrorMsg) {
	// Get tool from active tools
	tool, exists := m.activeTools[msg.executionID]
	if exists {
		// Remove from active tools
		delete(m.activeTools, msg.executionID)

		// Add error message
		m.messages = append(m.messages, Message{
			Type:      msgToolError,
			Content:   fmt.Sprintf("❌ %s: Tool error - %v", tool.ToolName, msg.err),
			AgentID:   tool.AgentID,
			Timestamp: time.Now(),
			Metadata: map[string]string{
				"execution_id": msg.executionID,
			},
		})
	} else {
		// Add error message without agent info
		m.messages = append(m.messages, Message{
			Type:      msgToolError,
			Content:   fmt.Sprintf("❌ Execution %s error - %v", msg.executionID, msg.err),
			Timestamp: time.Now(),
			Metadata: map[string]string{
				"execution_id": msg.executionID,
			},
		})
	}
	m.updateMessagesView()
}

func (m *ChatModel) handleToolAuthRequired(msg toolAuthRequiredMsg) {
	// Add auth required message
	m.messages = append(m.messages, Message{
		Type:      msgToolAuth,
		Content:   fmt.Sprintf("🔐 Authentication required for %s: %s", msg.toolName, msg.message),
		Timestamp: time.Now(),
		Metadata: map[string]string{
			"tool_name": msg.toolName,
			"auth_url":  msg.authURL,
		},
	})
	m.updateMessagesView()
}
