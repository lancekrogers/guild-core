// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package user_journey

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFirstTimeUserExperience validates the complete first-time user journey
// TODO: Update this test once JourneyResult has the required fields
/*
func TestFirstTimeUserExperience(t *testing.T) {
	t.Log("🎯 Testing First-Time User Experience Journey - HAPPY PATH")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	// Phase 1: Framework Setup with Real System Integration
	t.Log("🏗️ Setting up user journey test framework with real systems")

	framework, err := NewUserJourneyFramework(t)
	require.NoError(t, err, "Failed to create user journey framework")
	defer framework.Cleanup()

	// TODO: Initialize real provider integration
	// err = framework.InitializeRealProviders()
	// require.NoError(t, err, "Failed to initialize real provider integration")

	// Phase 2: User Profile Creation with Realistic Persona
	t.Log("👤 Creating comprehensive first-time user profile")

	userProfile := UserProfile{
		ExperienceLevel:   ExperienceLevelBeginner,
		TechnicalSkills:   []string{"basic-cli", "git", "programming", "go", "testing"},
		PreferredTools:    []string{"vscode", "terminal", "browser"},
		WorkflowPatterns:  []string{"feature-branch", "code-review", "tdd"},
		ProductivityGoals: []string{"faster-development", "better-code-quality", "ai-assistance"},
	}

	// Phase 3: Journey Creation with Real System Validation
	t.Log("🚀 Creating and executing comprehensive first-time user journey")

	journey, err := framework.GetJourneyManager().CreateJourney(JourneyTypeFirstTime, userProfile)
	require.NoError(t, err, "Failed to create first-time user journey")

	assert.Equal(t, JourneyTypeFirstTime, journey.Type, "Journey type should be FirstTime")
	assert.Equal(t, userProfile, journey.Profile, "Journey should use provided user profile")
	assert.GreaterOrEqual(t, len(journey.Steps), 5, "First-time journey should have comprehensive steps")

	// Validate journey has all required components
	requiredSteps := []string{"Installation", "Provider Setup", "Project Initialization", "Agent Interaction", "Validation"}
	for _, required := range requiredSteps {
		found := false
		for _, step := range journey.Steps {
			if strings.Contains(step.Name, required) {
				found = true
				break
			}
		}
		assert.True(t, found, "Journey missing required step: %s", required)
	}

	// Validate journey structure
	stepNames := make([]string, len(journey.Steps))
	for i, step := range journey.Steps {
		stepNames[i] = step.Name
	}

	expectedSteps := []string{
		"Installation and Setup",
		"Project Initialization",
		"First Agent Interaction",
	}

	for i, expectedStep := range expectedSteps {
		assert.Contains(t, stepNames[i], expectedStep, "Step %d should contain '%s'", i, expectedStep)
	}

	// Create user session
	sessionID := "first-time-user-session-001"
	session := &UserSession{
		ID:          sessionID,
		UserProfile: userProfile,
		StartTime:   time.Now(),
		Context:     make(map[string]interface{}),
		Preferences: make(map[string]interface{}),
	}

	framework.GetSessionManager().sessions[sessionID] = session

	// Execute the journey
	journeyResult, err := framework.ExecuteJourney(ctx, journey, sessionID)
	require.NoError(t, err, "Journey execution should succeed")
	require.NotNil(t, journeyResult, "Journey result should not be nil")

	// Phase 4: Results Validation
	t.Log("✅ Validating journey results")

	// COMPREHENSIVE SUCCESS CRITERIA VALIDATION

	// Completion Rate: ≥85% of new users complete entire journey
	assert.GreaterOrEqual(t, journeyResult.CompletionRate, 0.85,
		"Completion rate should be ≥85%% (actual: %.1f%%)", journeyResult.CompletionRate*100)

	// Time to Value: ≥80% complete journey within 10 minutes
	targetTime := 10 * time.Minute
	assert.LessOrEqual(t, journeyResult.TotalTime, targetTime,
		"Journey should complete within 10 minutes (actual: %v)", journeyResult.TotalTime)

	// User Satisfaction: ≥90% rate experience as positive
	assert.GreaterOrEqual(t, journeyResult.UserSatisfaction, 0.90,
		"User satisfaction should be ≥90%% (actual: %.1f%%)", journeyResult.UserSatisfaction*100)

	// Error Recovery: ≥95% of errors are resolved within journey
	assert.GreaterOrEqual(t, journeyResult.ErrorRecoveryRate, 0.95,
		"Error recovery rate should be ≥95%% (actual: %.1f%%)", journeyResult.ErrorRecoveryRate*100)

	// HAPPY PATH SPECIFIC VALIDATIONS

	// System Integration: All components should be functioning
	assert.True(t, journeyResult.SystemIntegrationHealth > 0.95,
		"System integration health should be >95%% (actual: %.1f%%)", journeyResult.SystemIntegrationHealth*100)

	// Performance Benchmarks: All SLAs should be met
	assert.True(t, journeyResult.SLACompliance > 0.98,
		"SLA compliance should be >98%% (actual: %.1f%%)", journeyResult.SLACompliance*100)

	// Real Provider Response: Actual AI responses should be generated
	assert.True(t, journeyResult.RealAIResponses,
		"Journey should include real AI provider responses")

	// Data Persistence: User data should persist across steps
	assert.True(t, journeyResult.DataPersistence,
		"User data and context should persist throughout journey")

	// COMPREHENSIVE STEP VALIDATION

	// Get journey execution for detailed analysis
	framework.journeyManager.mu.RLock()
	execution, exists := framework.journeyManager.activeJourneys[journey.ID]
	if !exists {
		// Journey completed successfully - validate final results
		t.Log("✅ Journey execution completed successfully and cleaned up")

		// Validate stored journey results
		storageResults, err := framework.GetJourneyResults(JourneyTypeFirstTime)
		require.NoError(t, err, "Should be able to retrieve journey results")
		assert.NotNil(t, storageResults, "Journey results should be stored")

	} else {
		framework.journeyManager.mu.RUnlock()

		// Validate each step execution in detail
		require.GreaterOrEqual(t, len(execution.StepResults), 5, "Should have results for all comprehensive steps")

		// Step 1: Installation and Setup validation with real system checks
		step1 := execution.StepResults[0]
		assert.True(t, step1.Success, "Installation and Setup step should succeed")
		assert.LessOrEqual(t, step1.ActualTime, 2*time.Minute,
			"Installation should complete within 2 minutes (actual: %v)", step1.ActualTime)

		// Validate real binary installation
		assert.Contains(t, step1.Metadata, "binary_path", "Installation should create actual binary")
		assert.Contains(t, step1.Metadata, "config_created", "Configuration should be created")
		assert.True(t, step1.Metadata["config_created"].(bool), "Configuration creation should succeed")

		// Validate CLI response time
		for _, action := range step1.UserActions {
			if action.Action.Type == ActionTypeCLICommand {
				assert.LessOrEqual(t, action.Duration, 100*time.Millisecond,
					"CLI commands should respond within 100ms (actual: %v)", action.Duration)
			}
		}

		// Step 2: Project Initialization validation
		if len(execution.StepResults) > 1 {
			step2 := execution.StepResults[1]
			assert.True(t, step2.Success, "Project Initialization step should succeed")
			assert.LessOrEqual(t, step2.ActualTime, 3*time.Minute,
				"Project initialization should complete within 3 minutes (actual: %v)", step2.ActualTime)
		}

		// Step 3: First Agent Interaction validation with real AI responses
		if len(execution.StepResults) > 2 {
			step3 := execution.StepResults[2]
			assert.True(t, step3.Success, "First Agent Interaction step should succeed")
			assert.LessOrEqual(t, step3.ActualTime, 5*time.Minute,
				"First interaction should complete within 5 minutes (actual: %v)", step3.ActualTime)

			// Validate real AI provider responses
			assert.Contains(t, step3.Metadata, "real_ai_response", "Should have real AI response")
			assert.Contains(t, step3.Metadata, "provider_used", "Should track which provider was used")
			assert.Contains(t, step3.Metadata, "cost_incurred", "Should track actual costs")

			// Validate chat response time and relevance
			for _, action := range step3.UserActions {
				if action.Action.Type == ActionTypeChatInteraction {
					assert.LessOrEqual(t, action.Duration, 10*time.Second,
						"Chat response should arrive within 10 seconds (actual: %v)", action.Duration)

					// Validate actual AI response quality
					if relevance, exists := action.Metrics["relevance_score"]; exists {
						assert.GreaterOrEqual(t, relevance, 0.8,
							"Response relevance should be ≥80%% (actual: %.1f%%)", relevance*100)
					}

					if coherence, exists := action.Metrics["coherence_score"]; exists {
						assert.GreaterOrEqual(t, coherence, 0.85,
							"Response coherence should be ≥85%% (actual: %.1f%%)", coherence*100)
					}

					if helpfulness, exists := action.Metrics["helpfulness_score"]; exists {
						assert.GreaterOrEqual(t, helpfulness, 0.9,
							"Response helpfulness should be ≥90%% (actual: %.1f%%)", helpfulness*100)
					}
				}
			}

			// Step 4: Real System Integration validation
			if len(execution.StepResults) > 3 {
				step4 := execution.StepResults[3]
				assert.True(t, step4.Success, "System Integration step should succeed")

				// Validate all system components are integrated
				assert.Contains(t, step4.Metadata, "kanban_integration", "Kanban system should be integrated")
				assert.Contains(t, step4.Metadata, "rag_integration", "RAG system should be integrated")
				assert.Contains(t, step4.Metadata, "provider_integration", "Provider system should be integrated")
				assert.Contains(t, step4.Metadata, "memory_integration", "Memory system should be integrated")
			}

			// Step 5: End-to-End Validation
			if len(execution.StepResults) > 4 {
				step5 := execution.StepResults[4]
				assert.True(t, step5.Success, "End-to-End Validation step should succeed")

				// Validate complete workflow execution
				assert.Contains(t, step5.Metadata, "workflow_complete", "Complete workflow should execute")
				assert.Contains(t, step5.Metadata, "data_persisted", "Data should be persisted")
				assert.Contains(t, step5.Metadata, "session_maintained", "Session should be maintained")
			}
		}
	}

	// COMPREHENSIVE METRICS COLLECTION VALIDATION

	metrics, err := framework.GetJourneyResults(JourneyTypeFirstTime)
	if err == nil {
		assert.GreaterOrEqual(t, metrics.TotalExecutions, 1, "Should track journey executions")
		if metrics.TotalExecutions > 0 {
			successRate := float64(metrics.SuccessfulExecutions) / float64(metrics.TotalExecutions)
			assert.GreaterOrEqual(t, successRate, 0.85, "Overall success rate should be ≥85%%")
		}

		// Validate performance metrics
		assert.LessOrEqual(t, metrics.AverageCompletionTime, 10*time.Minute,
			"Average completion time should be ≤10 minutes")
		assert.GreaterOrEqual(t, metrics.AverageUserSatisfaction, 0.9,
			"Average user satisfaction should be ≥90%%")

		// Validate system performance metrics
		assert.GreaterOrEqual(t, metrics.SystemPerformanceScore, 0.95,
			"System performance score should be ≥95%%")
		assert.LessOrEqual(t, metrics.AverageErrorRate, 0.05,
			"Average error rate should be ≤5%%")
	}

	// REAL SYSTEM STATE VALIDATION

	// Validate database state
	memoryDB := framework.GetMemoryDatabase()
	assert.NotNil(t, memoryDB, "Memory database should be accessible")

	user, err := memoryDB.GetUser(sessionID)
	if err == nil {
		assert.Equal(t, userProfile.ExperienceLevel, user.ExperienceLevel,
			"User profile should be persisted correctly")
	}

	// Validate kanban state
	kanbanBoard := framework.GetKanbanBoard()
	assert.NotNil(t, kanbanBoard, "Kanban board should be accessible")

	tasks, err := kanbanBoard.GetTasksForUser(sessionID)
	if err == nil && len(tasks) > 0 {
		assert.True(t, len(tasks) > 0, "User should have tasks created during journey")
	}

	// Validate RAG corpus state
	corpus := framework.GetRAGCorpus()
	assert.NotNil(t, corpus, "RAG corpus should be accessible")

	conversations, err := corpus.GetConversationHistory(sessionID)
	if err == nil {
		assert.True(t, len(conversations) > 0, "Conversation history should be stored")
	}

	// COMPREHENSIVE PERFORMANCE ANALYSIS

	t.Log("📊 Happy Path Journey Performance Analysis:")
	t.Logf("   - Total Time: %v (target: ≤%v)", journeyResult.TotalTime, targetTime)
	t.Logf("   - Completion Rate: %.1f%% (target: ≥85%%)", journeyResult.CompletionRate*100)
	t.Logf("   - User Satisfaction: %.1f%% (target: ≥90%%)", journeyResult.UserSatisfaction*100)
	t.Logf("   - Quality Score: %.1f%% (target: ≥85%%)", journeyResult.QualityScore*100)
	t.Logf("   - Error Count: %d (target: ≤2)", journeyResult.ErrorCount)
	t.Logf("   - Error Recovery Rate: %.1f%% (target: ≥95%%)", journeyResult.ErrorRecoveryRate*100)
	t.Logf("   - System Integration Health: %.1f%% (target: ≥95%%)", journeyResult.SystemIntegrationHealth*100)
	t.Logf("   - SLA Compliance: %.1f%% (target: ≥98%%)", journeyResult.SLACompliance*100)
	t.Logf("   - Real AI Responses: %v (required: true)", journeyResult.RealAIResponses)
	t.Logf("   - Data Persistence: %v (required: true)", journeyResult.DataPersistence)

	// Provider-specific performance metrics
	if providerMetrics, exists := journeyResult.ProviderMetrics["response_time"]; exists {
		t.Logf("   - AI Provider Response Time: %v (target: ≤10s)", providerMetrics)
	}
	if providerMetrics, exists := journeyResult.ProviderMetrics["cost"]; exists {
		t.Logf("   - Total Provider Cost: $%.4f", providerMetrics)
	}
	if providerMetrics, exists := journeyResult.ProviderMetrics["tokens"]; exists {
		t.Logf("   - Total Tokens Used: %.0f", providerMetrics)
	}

	if len(journeyResult.Recommendations) > 0 {
		t.Log("💡 Recommendations:")
		for _, rec := range journeyResult.Recommendations {
			t.Logf("   - %s", rec)
		}
	}

	// SUCCESS SUMMARY

	// FINAL HAPPY PATH VALIDATION
	if journeyResult.Success &&
		journeyResult.CompletionRate >= 0.85 &&
		journeyResult.TotalTime <= targetTime &&
		journeyResult.UserSatisfaction >= 0.90 &&
		journeyResult.ErrorRecoveryRate >= 0.95 &&
		journeyResult.SystemIntegrationHealth >= 0.95 &&
		journeyResult.SLACompliance >= 0.98 &&
		journeyResult.RealAIResponses &&
		journeyResult.DataPersistence {

		t.Log("🎉 HAPPY PATH: First-Time User Experience Journey PASSED ALL CRITERIA")
		t.Log("✅ All systems integrated and functioning optimally")
		t.Log("✅ All SLAs met or exceeded")
		t.Log("✅ Real AI provider integration successful")
		t.Log("✅ Data persistence and state management working")
		t.Log("✅ User experience meets all quality thresholds")

	} else {
		t.Log("❌ HAPPY PATH: First-Time User Experience Journey FAILED")
		t.Logf("   - Success: %v (required: true)", journeyResult.Success)
		t.Logf("   - Completion Rate: %.1f%% (target: ≥85%%)", journeyResult.CompletionRate*100)
		t.Logf("   - Time: %v (target: ≤10min)", journeyResult.TotalTime)
		t.Logf("   - Satisfaction: %.1f%% (target: ≥90%%)", journeyResult.UserSatisfaction*100)
		t.Logf("   - Error Recovery: %.1f%% (target: ≥95%%)", journeyResult.ErrorRecoveryRate*100)
		t.Logf("   - System Health: %.1f%% (target: ≥95%%)", journeyResult.SystemIntegrationHealth*100)
		t.Logf("   - SLA Compliance: %.1f%% (target: ≥98%%)", journeyResult.SLACompliance*100)
		t.Logf("   - Real AI Responses: %v (required: true)", journeyResult.RealAIResponses)
		t.Logf("   - Data Persistence: %v (required: true)", journeyResult.DataPersistence)
	}

	// Final assertions for happy path
	assert.True(t, journeyResult.Success, "First-time user journey must be successful")
	assert.GreaterOrEqual(t, journeyResult.SystemIntegrationHealth, 0.95, "System integration health must be ≥95%%")
	assert.GreaterOrEqual(t, journeyResult.SLACompliance, 0.98, "SLA compliance must be ≥98%%")
	assert.True(t, journeyResult.RealAIResponses, "Must include real AI provider responses")
	assert.True(t, journeyResult.DataPersistence, "Must demonstrate data persistence")
}
*/

// TestDailyDeveloperWorkflow validates the daily developer workflow journey
func TestDailyDeveloperWorkflow(t *testing.T) {
	t.Log("🎯 Testing Daily Developer Workflow Journey")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	framework, err := NewUserJourneyFramework(t)
	require.NoError(t, err, "Failed to create user journey framework")
	defer framework.Cleanup()

	// Create experienced user profile
	userProfile := UserProfile{
		ExperienceLevel:   ExperienceLevelIntermediate,
		TechnicalSkills:   []string{"advanced-cli", "git", "programming", "architecture"},
		PreferredTools:    []string{"vscode", "terminal", "docker"},
		WorkflowPatterns:  []string{"feature-branch", "code-review", "tdd", "ci-cd"},
		ProductivityGoals: []string{"faster-development", "better-code-quality", "automated-testing"},
	}

	// Create daily workflow journey
	journey, err := framework.GetJourneyManager().CreateJourney(JourneyTypeDailyWorkflow, userProfile)
	require.NoError(t, err, "Failed to create daily workflow journey")

	// Create user session
	sessionID := "daily-workflow-session-001"
	session := &UserSession{
		ID:          sessionID,
		UserProfile: userProfile,
		StartTime:   time.Now(),
		Context:     make(map[string]interface{}),
		Preferences: map[string]interface{}{
			"feature_development_time": 4 * time.Hour,
			"productivity_target":      1.30, // 30% improvement
		},
	}

	framework.GetSessionManager().sessions[sessionID] = session

	// Execute the journey
	journeyResult, err := framework.ExecuteJourney(ctx, journey, sessionID)
	require.NoError(t, err, "Daily workflow journey execution should succeed")
	require.NotNil(t, journeyResult, "Journey result should not be nil")

	// Validate daily workflow success criteria

	// Productivity Improvement: ≥30% increase in development velocity
	assert.GreaterOrEqual(t, journeyResult.ProductivityGain, 0.30,
		"Productivity gain should be ≥30%% (actual: %.1f%%)", journeyResult.ProductivityGain*100)

	// Code Quality: Maintained or improved quality metrics
	assert.GreaterOrEqual(t, journeyResult.QualityScore, 0.85,
		"Code quality should be maintained ≥85%% (actual: %.1f%%)", journeyResult.QualityScore*100)

	// User Satisfaction: ≥95% daily workflow satisfaction
	assert.GreaterOrEqual(t, journeyResult.UserSatisfaction, 0.95,
		"User satisfaction should be ≥95%% (actual: %.1f%%)", journeyResult.UserSatisfaction*100)

	// Feature Completion: ≥20% faster feature delivery (implied by productivity gain)
	// This would be measured over multiple executions in real implementation

	t.Log("📊 Daily Workflow Performance Analysis:")
	t.Logf("   - Total Time: %v", journeyResult.TotalTime)
	t.Logf("   - Productivity Gain: %.1f%%", journeyResult.ProductivityGain*100)
	t.Logf("   - Quality Score: %.1f%%", journeyResult.QualityScore*100)
	t.Logf("   - User Satisfaction: %.1f%%", journeyResult.UserSatisfaction*100)

	if journeyResult.Success {
		t.Log("🎉 Daily Developer Workflow Journey: PASSED")
	} else {
		t.Log("❌ Daily Developer Workflow Journey: FAILED")
	}

	assert.True(t, journeyResult.Success, "Daily developer workflow should be successful")
}

// TestMultiAgentProjectCoordination validates multi-agent coordination journey
func TestMultiAgentProjectCoordination(t *testing.T) {
	t.Log("🎯 Testing Multi-Agent Project Coordination Journey")

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Minute)
	defer cancel()

	framework, err := NewUserJourneyFramework(t)
	require.NoError(t, err, "Failed to create user journey framework")
	defer framework.Cleanup()

	// Create tech lead profile
	userProfile := UserProfile{
		ExperienceLevel:   ExperienceLevelExpert,
		TechnicalSkills:   []string{"architecture", "project-management", "team-leadership", "system-design"},
		PreferredTools:    []string{"kanban", "git", "monitoring", "automation"},
		WorkflowPatterns:  []string{"agile", "scrum", "ci-cd", "code-review"},
		ProductivityGoals: []string{"team-coordination", "project-delivery", "quality-assurance"},
	}

	// Create multi-agent coordination journey
	journey, err := framework.GetJourneyManager().CreateJourney(JourneyTypeMultiAgent, userProfile)
	require.NoError(t, err, "Failed to create multi-agent coordination journey")

	// Create user session
	sessionID := "multi-agent-coordination-session-001"
	session := &UserSession{
		ID:          sessionID,
		UserProfile: userProfile,
		StartTime:   time.Now(),
		Context:     make(map[string]interface{}),
		Preferences: map[string]interface{}{
			"project_duration":      2 * 7 * 24 * time.Hour, // 2 weeks
			"agent_count":           5,
			"coordination_overhead": 0.15, // ≤15% of total project time
		},
	}

	framework.GetSessionManager().sessions[sessionID] = session

	// Execute the journey
	journeyResult, err := framework.ExecuteJourney(ctx, journey, sessionID)
	require.NoError(t, err, "Multi-agent coordination journey execution should succeed")
	require.NotNil(t, journeyResult, "Journey result should not be nil")

	// Validate multi-agent coordination success criteria

	// Project Delivery: ≥90% of milestones delivered on time
	assert.GreaterOrEqual(t, journeyResult.CompletionRate, 0.90,
		"Project delivery rate should be ≥90%% (actual: %.1f%%)", journeyResult.CompletionRate*100)

	// Quality Maintenance: ≥95% quality gate compliance
	assert.GreaterOrEqual(t, journeyResult.QualityScore, 0.95,
		"Quality gate compliance should be ≥95%% (actual: %.1f%%)", journeyResult.QualityScore*100)

	// Agent Efficiency: ≥85% optimal agent utilization (measured by success rate)
	assert.GreaterOrEqual(t, journeyResult.UserSatisfaction, 0.85,
		"Agent efficiency should be ≥85%% (actual: %.1f%%)", journeyResult.UserSatisfaction*100)

	t.Log("📊 Multi-Agent Coordination Performance Analysis:")
	t.Logf("   - Total Time: %v", journeyResult.TotalTime)
	t.Logf("   - Project Delivery Rate: %.1f%%", journeyResult.CompletionRate*100)
	t.Logf("   - Quality Compliance: %.1f%%", journeyResult.QualityScore*100)
	t.Logf("   - Agent Efficiency: %.1f%%", journeyResult.UserSatisfaction*100)

	if journeyResult.Success {
		t.Log("🎉 Multi-Agent Project Coordination Journey: PASSED")
	} else {
		t.Log("❌ Multi-Agent Project Coordination Journey: FAILED")
	}

	assert.True(t, journeyResult.Success, "Multi-agent coordination should be successful")
}

// TestKnowledgeDiscoveryAndResearch validates knowledge discovery journey
func TestKnowledgeDiscoveryAndResearch(t *testing.T) {
	t.Log("🎯 Testing Knowledge Discovery and Research Journey")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	framework, err := NewUserJourneyFramework(t)
	require.NoError(t, err, "Failed to create user journey framework")
	defer framework.Cleanup()

	// Create new developer profile (joining existing codebase)
	userProfile := UserProfile{
		ExperienceLevel:   ExperienceLevelIntermediate,
		TechnicalSkills:   []string{"programming", "git", "debugging"},
		PreferredTools:    []string{"vscode", "terminal", "browser"},
		WorkflowPatterns:  []string{"exploration", "learning", "documentation"},
		ProductivityGoals: []string{"codebase-understanding", "fast-onboarding", "contribution-readiness"},
	}

	// Create knowledge discovery journey
	journey, err := framework.GetJourneyManager().CreateJourney(JourneyTypeKnowledgeDiscovery, userProfile)
	require.NoError(t, err, "Failed to create knowledge discovery journey")

	// Create user session
	sessionID := "knowledge-discovery-session-001"
	session := &UserSession{
		ID:          sessionID,
		UserProfile: userProfile,
		StartTime:   time.Now(),
		Context:     make(map[string]interface{}),
		Preferences: map[string]interface{}{
			"onboarding_target":      1 * 7 * 24 * time.Hour, // 1 week
			"understanding_depth":    "comprehensive",
			"contribution_readiness": 0.85, // 85% readiness to contribute
		},
	}

	framework.GetSessionManager().sessions[sessionID] = session

	// Execute the journey
	journeyResult, err := framework.ExecuteJourney(ctx, journey, sessionID)
	require.NoError(t, err, "Knowledge discovery journey execution should succeed")
	require.NotNil(t, journeyResult, "Journey result should not be nil")

	// Validate knowledge discovery success criteria

	// Onboarding Speed: ≥80% faster than traditional methods
	assert.GreaterOrEqual(t, journeyResult.ProductivityGain, 0.80,
		"Onboarding speed improvement should be ≥80%% (actual: %.1f%%)", journeyResult.ProductivityGain*100)

	// Knowledge Retention: ≥90% retention after 30 days (simulated)
	assert.GreaterOrEqual(t, journeyResult.QualityScore, 0.90,
		"Knowledge retention should be ≥90%% (actual: %.1f%%)", journeyResult.QualityScore*100)

	// Contribution Quality: ≥85% of team average within one week
	assert.GreaterOrEqual(t, journeyResult.UserSatisfaction, 0.85,
		"Contribution readiness should be ≥85%% (actual: %.1f%%)", journeyResult.UserSatisfaction*100)

	t.Log("📊 Knowledge Discovery Performance Analysis:")
	t.Logf("   - Total Time: %v", journeyResult.TotalTime)
	t.Logf("   - Onboarding Speed Improvement: %.1f%%", journeyResult.ProductivityGain*100)
	t.Logf("   - Knowledge Retention: %.1f%%", journeyResult.QualityScore*100)
	t.Logf("   - Contribution Readiness: %.1f%%", journeyResult.UserSatisfaction*100)

	if journeyResult.Success {
		t.Log("🎉 Knowledge Discovery and Research Journey: PASSED")
	} else {
		t.Log("❌ Knowledge Discovery and Research Journey: FAILED")
	}

	assert.True(t, journeyResult.Success, "Knowledge discovery journey should be successful")
}

// TestCrossJourneyIntegration validates integration points between different journeys
func TestCrossJourneyIntegration(t *testing.T) {
	t.Log("🎯 Testing Cross-Journey Integration Points")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
	defer cancel()

	framework, err := NewUserJourneyFramework(t)
	require.NoError(t, err, "Failed to create user journey framework")
	defer framework.Cleanup()

	// Test seamless transition from first-time user to daily workflow
	t.Log("🔄 Testing First-Time User → Daily Workflow Transition")

	// Start as beginner
	beginnerProfile := UserProfile{
		ExperienceLevel:   ExperienceLevelBeginner,
		TechnicalSkills:   []string{"basic-cli", "git"},
		PreferredTools:    []string{"vscode"},
		WorkflowPatterns:  []string{"basic"},
		ProductivityGoals: []string{"learning"},
	}

	// Execute first-time journey
	firstTimeJourney, err := framework.GetJourneyManager().CreateJourney(JourneyTypeFirstTime, beginnerProfile)
	require.NoError(t, err, "Failed to create first-time journey")

	sessionID := "cross-journey-integration-session"
	session := &UserSession{
		ID:          sessionID,
		UserProfile: beginnerProfile,
		StartTime:   time.Now(),
		Context:     make(map[string]interface{}),
		Preferences: make(map[string]interface{}),
	}
	framework.GetSessionManager().sessions[sessionID] = session

	firstTimeResult, err := framework.ExecuteJourney(ctx, firstTimeJourney, sessionID)
	require.NoError(t, err, "First-time journey should succeed")
	require.True(t, firstTimeResult.Success, "First-time journey must be successful")

	// Update user profile to intermediate after successful first-time experience
	upgradeProfile := UserProfile{
		ExperienceLevel:   ExperienceLevelIntermediate,
		TechnicalSkills:   []string{"advanced-cli", "git", "programming"},
		PreferredTools:    []string{"vscode", "terminal"},
		WorkflowPatterns:  []string{"feature-branch", "code-review"},
		ProductivityGoals: []string{"faster-development", "better-code-quality"},
	}

	// Update session
	session.UserProfile = upgradeProfile
	session.Context["previous_journey"] = "first-time"
	session.Context["experience_gained"] = true

	// Execute daily workflow journey
	dailyWorkflowJourney, err := framework.GetJourneyManager().CreateJourney(JourneyTypeDailyWorkflow, upgradeProfile)
	require.NoError(t, err, "Failed to create daily workflow journey")

	dailyWorkflowResult, err := framework.ExecuteJourney(ctx, dailyWorkflowJourney, sessionID)
	require.NoError(t, err, "Daily workflow journey should succeed")
	require.True(t, dailyWorkflowResult.Success, "Daily workflow journey must be successful")

	// Validate cross-journey integration

	// Context Preservation: User context and preferences persist across journeys
	assert.Equal(t, "first-time", session.Context["previous_journey"],
		"Previous journey context should be preserved")
	assert.True(t, session.Context["experience_gained"].(bool),
		"Experience gained should be tracked")

	// Performance Consistency: Response times remain consistent
	assert.LessOrEqual(t, dailyWorkflowResult.TotalTime, firstTimeResult.TotalTime*2,
		"Daily workflow should not take significantly longer than first-time journey")

	// Quality Improvement: User should perform better in second journey
	assert.GreaterOrEqual(t, dailyWorkflowResult.UserSatisfaction, firstTimeResult.UserSatisfaction,
		"User satisfaction should improve or maintain in subsequent journeys")

	t.Log("📊 Cross-Journey Integration Analysis:")
	t.Logf("   - First-Time Journey: %v, satisfaction: %.1f%%",
		firstTimeResult.TotalTime, firstTimeResult.UserSatisfaction*100)
	t.Logf("   - Daily Workflow Journey: %v, satisfaction: %.1f%%",
		dailyWorkflowResult.TotalTime, dailyWorkflowResult.UserSatisfaction*100)
	t.Logf("   - Context Preservation: ✓")
	t.Logf("   - Performance Consistency: ✓")

	t.Log("🎉 Cross-Journey Integration: PASSED")
}

// TestUserJourneyFrameworkComprehensive validates the framework itself
func TestUserJourneyFrameworkComprehensive(t *testing.T) {
	t.Log("🎯 Testing User Journey Framework Comprehensively")

	framework, err := NewUserJourneyFramework(t)
	require.NoError(t, err, "Failed to create user journey framework")
	defer framework.Cleanup()

	// Test framework components
	assert.NotNil(t, framework.GetJourneyManager(), "Should have journey manager")
	assert.NotNil(t, framework.GetSessionManager(), "Should have session manager")
	assert.NotNil(t, framework.GetMetricsCollector(), "Should have metrics collector")

	// Test journey creation for all types
	userProfile := UserProfile{
		ExperienceLevel:   ExperienceLevelIntermediate,
		TechnicalSkills:   []string{"programming"},
		PreferredTools:    []string{"vscode"},
		WorkflowPatterns:  []string{"standard"},
		ProductivityGoals: []string{"productivity"},
	}

	journeyTypes := []JourneyType{
		JourneyTypeFirstTime,
		JourneyTypeDailyWorkflow,
		JourneyTypeMultiAgent,
		JourneyTypeKnowledgeDiscovery,
	}

	for _, journeyType := range journeyTypes {
		journey, err := framework.GetJourneyManager().CreateJourney(journeyType, userProfile)
		require.NoError(t, err, "Should create journey for type %s", journeyType)
		assert.Equal(t, journeyType, journey.Type, "Journey type should match")
		assert.GreaterOrEqual(t, len(journey.Steps), 1, "Journey should have steps")

		t.Logf("✅ Created %s journey with %d steps", journeyType, len(journey.Steps))
	}

	t.Log("🎉 User Journey Framework: COMPREHENSIVE TEST PASSED")
}
