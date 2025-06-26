// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package elena

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPlanningDialogue(t *testing.T) {
	dialogue := NewPlanningDialogue("test-dialogue-123")
	
	assert.NotNil(t, dialogue)
	assert.Equal(t, "test-dialogue-123", dialogue.ID)
	assert.Equal(t, StageIntroduction, dialogue.stage)
	assert.NotNil(t, dialogue.context)
	assert.NotNil(t, dialogue.responses)
}

func TestGetNextQuestion_AllStages(t *testing.T) {
	tests := []struct {
		name     string
		stage    PlanningStage
		contains string
	}{
		{
			name:     "introduction",
			stage:    StageIntroduction,
			contains: "Greetings, noble artisan",
		},
		{
			name:     "project_purpose",
			stage:    StageProjectPurpose,
			contains: "Build software",
		},
		{
			name:     "project_type",
			stage:    StageProjectType,
			contains: "specific type",
		},
		{
			name:     "technology",
			stage:    StageTechnology,
			contains: "tools of thy craft",
		},
		{
			name:     "requirements",
			stage:    StageRequirements,
			contains: "specific requirements",
		},
		{
			name:     "constraints",
			stage:    StageConstraints,
			contains: "challenges and boundaries",
		},
		{
			name:     "team_size",
			stage:    StageTeamSize,
			contains: "thy company",
		},
		{
			name:     "timeline",
			stage:    StageTimeline,
			contains: "timeline",
		},
		{
			name:     "summary",
			stage:    StageSummary,
			contains: "Commission Summary",
		},
		{
			name:     "complete",
			stage:    StageComplete,
			contains: "commission planning is complete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dialogue := NewPlanningDialogue("test")
			dialogue.stage = tt.stage
			
			// Add some test responses for summary stage
			if tt.stage == StageSummary {
				dialogue.responses["project_description"] = "Test project"
				dialogue.responses["project_purpose"] = "Build software"
				dialogue.responses["project_type"] = "API"
				dialogue.responses["team_size"] = "Small team"
				dialogue.responses["timeline"] = "1 month"
			}
			
			question := dialogue.GetNextQuestion()
			assert.Contains(t, question, tt.contains)
		})
	}
}

func TestProcessResponse_StageTransitions(t *testing.T) {
	ctx := context.Background()
	dialogue := NewPlanningDialogue("test")
	
	// Test introduction -> project purpose
	assert.Equal(t, StageIntroduction, dialogue.stage)
	err := dialogue.ProcessResponse(ctx, "Hello")
	require.NoError(t, err)
	assert.Equal(t, StageProjectPurpose, dialogue.stage)
	
	// Test project purpose -> project type
	err = dialogue.ProcessResponse(ctx, "Build software")
	require.NoError(t, err)
	assert.Equal(t, StageProjectType, dialogue.stage)
	assert.Equal(t, "build_software", dialogue.context["purpose_category"])
	
	// Test project type -> technology
	err = dialogue.ProcessResponse(ctx, "REST API")
	require.NoError(t, err)
	assert.Equal(t, StageTechnology, dialogue.stage)
	
	// Continue through remaining stages
	stages := []PlanningStage{
		StageRequirements,
		StageConstraints,
		StageTeamSize,
		StageTimeline,
		StageSummary,
	}
	
	for _, expectedStage := range stages {
		err = dialogue.ProcessResponse(ctx, "Test response")
		require.NoError(t, err)
		assert.Equal(t, expectedStage, dialogue.stage)
	}
	
	// Test summary -> complete
	err = dialogue.ProcessResponse(ctx, "yes")
	require.NoError(t, err)
	assert.Equal(t, StageComplete, dialogue.stage)
}

func TestProcessResponse_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	
	dialogue := NewPlanningDialogue("test")
	err := dialogue.ProcessResponse(ctx, "any response")
	
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context cancelled")
}

func TestDynamicQuestions_APIProject(t *testing.T) {
	dialogue := NewPlanningDialogue("test")
	dialogue.stage = StageProjectType
	dialogue.responses["project_purpose"] = "Build software"
	dialogue.responses["project_description"] = "I want to build a REST API for managing tasks"
	
	question := dialogue.GetNextQuestion()
	assert.Contains(t, question, "API")
	assert.Contains(t, question, "REST")
	assert.Contains(t, question, "GraphQL")
}

func TestDynamicQuestions_WebProject(t *testing.T) {
	dialogue := NewPlanningDialogue("test")
	dialogue.stage = StageProjectType
	dialogue.responses["project_purpose"] = "Build software"
	dialogue.responses["project_description"] = "I need a web application for e-commerce"
	
	question := dialogue.GetNextQuestion()
	assert.Contains(t, question, "web application")
	assert.Contains(t, question, "Frontend")
	assert.Contains(t, question, "Backend")
}

func TestDynamicQuestions_ResearchProject(t *testing.T) {
	dialogue := NewPlanningDialogue("test")
	dialogue.stage = StageProjectType
	dialogue.responses["project_purpose"] = "Conduct deep research"
	dialogue.responses["project_description"] = "Research microservices patterns"
	
	question := dialogue.GetNextQuestion()
	assert.Contains(t, question, "research")
	assert.Contains(t, question, "investigation")
}

func TestGetResponses(t *testing.T) {
	dialogue := NewPlanningDialogue("test")
	ctx := context.Background()
	
	// Add some responses
	dialogue.ProcessResponse(ctx, "Introduction response")
	dialogue.ProcessResponse(ctx, "Build software")
	
	responses := dialogue.GetResponses()
	assert.NotEmpty(t, responses)
	assert.Equal(t, "Introduction response", responses["introduction"])
	assert.Equal(t, "Build software", responses["project_purpose"])
}

func TestIsComplete(t *testing.T) {
	dialogue := NewPlanningDialogue("test")
	
	assert.False(t, dialogue.IsComplete())
	
	dialogue.stage = StageComplete
	assert.True(t, dialogue.IsComplete())
}

func TestSetProjectContext(t *testing.T) {
	dialogue := NewPlanningDialogue("test")
	
	dialogue.SetProjectContext("detected_technology", "Go")
	dialogue.SetProjectContext("project_path", "/home/user/project")
	
	assert.Equal(t, "Go", dialogue.context["detected_technology"])
	assert.Equal(t, "/home/user/project", dialogue.context["project_path"])
}

func TestSummaryEditHandling(t *testing.T) {
	ctx := context.Background()
	dialogue := NewPlanningDialogue("test")
	
	// Set up dialogue in summary stage
	dialogue.stage = StageSummary
	dialogue.responses["project_description"] = "Test project"
	dialogue.responses["project_purpose"] = "Build software"
	dialogue.responses["project_type"] = "API"
	dialogue.responses["team_size"] = "Small team"
	dialogue.responses["timeline"] = "1 month"
	
	// Test editing technology
	err := dialogue.ProcessResponse(ctx, "change technology")
	require.NoError(t, err)
	assert.Equal(t, StageTechnology, dialogue.stage)
	
	// Reset to summary
	dialogue.stage = StageSummary
	
	// Test editing requirements
	err = dialogue.ProcessResponse(ctx, "update requirements")
	require.NoError(t, err)
	assert.Equal(t, StageRequirements, dialogue.stage)
}

func TestFormatHelpers(t *testing.T) {
	dialogue := NewPlanningDialogue("test")
	
	// Test formatList
	input := "item1\n\nitem2\n- item3\n* item4"
	formatted := dialogue.formatList(input)
	
	lines := strings.Split(formatted, "\n")
	assert.Equal(t, "- item1", lines[0])
	assert.Equal(t, "- item2", lines[1])
	assert.Equal(t, "- item3", lines[2])
	assert.Equal(t, "* item4", lines[3])
}

func TestPurposeProcessing(t *testing.T) {
	tests := []struct {
		name             string
		response         string
		expectedCategory string
	}{
		{
			name:             "build_software_explicit",
			response:         "1",
			expectedCategory: "build_software",
		},
		{
			name:             "build_software_text",
			response:         "I want to build software",
			expectedCategory: "build_software",
		},
		{
			name:             "research_explicit",
			response:         "2",
			expectedCategory: "deep_research",
		},
		{
			name:             "research_text",
			response:         "conduct research on systems",
			expectedCategory: "deep_research",
		},
		{
			name:             "improve_explicit",
			response:         "3",
			expectedCategory: "improve_existing",
		},
		{
			name:             "other",
			response:         "something completely different",
			expectedCategory: "other",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dialogue := NewPlanningDialogue("test")
			dialogue.processPurpose(tt.response)
			assert.Equal(t, tt.expectedCategory, dialogue.context["purpose_category"])
		})
	}
}