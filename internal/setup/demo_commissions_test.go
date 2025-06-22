// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package setup

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDemoCommissionGenerator(t *testing.T) {
	generator := NewDemoCommissionGenerator()
	assert.NotNil(t, generator)
	assert.NotNil(t, generator.templates)
	assert.Greater(t, len(generator.templates), 0)
}

func TestGenerateCommission(t *testing.T) {
	generator := NewDemoCommissionGenerator()
	ctx := context.Background()

	tests := []struct {
		name     string
		demoType DemoCommissionType
		wantErr  bool
		checkFor []string // Strings that should be in the content
	}{
		{
			name:     "Generate API Service Demo",
			demoType: DemoTypeAPIService,
			wantErr:  false,
			checkFor: []string{
				"RESTful Task Management API",
				"JWT",
				"PostgreSQL",
				"OpenAPI",
				"/auth/register",
				"unit tests",
			},
		},
		{
			name:     "Generate Web App Demo",
			demoType: DemoTypeWebApp,
			wantErr:  false,
			checkFor: []string{
				"Modern Analytics Dashboard",
				"React",
				"TypeScript",
				"Real-time",
				"WebSockets",
				"Responsive Design",
			},
		},
		{
			name:     "Generate CLI Tool Demo",
			demoType: DemoTypeCLITool,
			wantErr:  false,
			checkFor: []string{
				"Developer Productivity CLI Tool",
				"DevFlow",
				"Cobra",
				"Bubble Tea",
				"Project Scaffolding",
				"devflow init",
			},
		},
		{
			name:     "Generate Data Analysis Demo",
			demoType: DemoTypeDataAnalysis,
			wantErr:  false,
			checkFor: []string{
				"Data Pipeline",
				"Analytics Platform",
				"Apache Airflow",
				"Kafka",
				"Machine Learning",
				"ETL",
			},
		},
		{
			name:     "Generate Microservices Demo",
			demoType: DemoTypeMicroservices,
			wantErr:  false,
			checkFor: []string{
				"Microservices E-commerce Platform",
				"Authentication Service",
				"Kubernetes",
				"gRPC",
				"Service Mesh",
				"distributed systems", // lowercase to match actual content
			},
		},
		{
			name:     "Generate AI Demo",
			demoType: DemoTypeAI,
			wantErr:  false,
			checkFor: []string{
				"Intelligent Content Recommendation",
				"PyTorch",
				"NLP",
				"Collaborative Filtering",
				"Feature Store",
				"model serving", // lowercase to match actual content
			},
		},
		{
			name:     "Generate Default Demo",
			demoType: DemoTypeDefault,
			wantErr:  false,
			checkFor: []string{
				"Simple API Development Task",
				"REST API",
				"CRUD operations",
				"tasks",
				"unit tests",
			},
		},
		{
			name:     "Unknown Demo Type Falls Back to Default",
			demoType: DemoCommissionType("unknown"),
			wantErr:  false,
			checkFor: []string{
				"Simple API Development Task",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := generator.GenerateCommission(ctx, tt.demoType)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, content)

				// Check that all expected strings are present
				for _, expected := range tt.checkFor {
					assert.Contains(t, content, expected, "Content should contain: %s", expected)
				}

				// Verify it's valid markdown
				assert.True(t, strings.HasPrefix(content, "#"), "Content should start with markdown header")
				assert.Contains(t, content, "## ", "Content should have section headers")
			}
		})
	}
}

func TestGenerateCommissionWithCancellation(t *testing.T) {
	generator := NewDemoCommissionGenerator()

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := generator.GenerateCommission(ctx, DemoTypeAPIService)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cancelled")
}

func TestGetAvailableTypes(t *testing.T) {
	generator := NewDemoCommissionGenerator()

	types := generator.GetAvailableTypes()
	assert.NotEmpty(t, types)

	// Should not include default in the list
	for _, demoType := range types {
		assert.NotEqual(t, DemoTypeDefault, demoType)
	}

	// Should include all main types
	expectedTypes := []DemoCommissionType{
		DemoTypeAPIService,
		DemoTypeWebApp,
		DemoTypeCLITool,
		DemoTypeDataAnalysis,
		DemoTypeMicroservices,
		DemoTypeAI,
	}

	for _, expected := range expectedTypes {
		found := false
		for _, actual := range types {
			if actual == expected {
				found = true
				break
			}
		}
		assert.True(t, found, "Should include type: %s", expected)
	}
}

func TestGetDemoDescription(t *testing.T) {
	generator := NewDemoCommissionGenerator()

	tests := []struct {
		demoType DemoCommissionType
		contains string
	}{
		{DemoTypeAPIService, "REST API"},
		{DemoTypeWebApp, "responsive web"},
		{DemoTypeCLITool, "CLI tool"},
		{DemoTypeDataAnalysis, "data processing"},
		{DemoTypeMicroservices, "microservices"},
		{DemoTypeAI, "AI-powered"},
		{DemoCommissionType("unknown"), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.demoType), func(t *testing.T) {
			desc := generator.GetDemoDescription(tt.demoType)
			assert.Contains(t, desc, tt.contains)
		})
	}
}

func TestInferDemoType(t *testing.T) {
	generator := NewDemoCommissionGenerator()
	ctx := context.Background()

	// Currently returns default, but test the interface
	demoType, err := generator.InferDemoType(ctx, "/test/project")
	assert.NoError(t, err)
	assert.Equal(t, DemoTypeDefault, demoType)
}

func TestInferDemoTypeWithCancellation(t *testing.T) {
	generator := NewDemoCommissionGenerator()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := generator.InferDemoType(ctx, "/test/project")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cancelled")
}

func TestGetRecommendedDemo(t *testing.T) {
	generator := NewDemoCommissionGenerator()
	ctx := context.Background()

	tests := []struct {
		name         string
		projectInfo  map[string]interface{}
		expectedType DemoCommissionType
		reasonCheck  string
	}{
		{
			name: "API project name",
			projectInfo: map[string]interface{}{
				"project_name": "user-api-service",
			},
			expectedType: DemoTypeAPIService,
			reasonCheck:  "API service",
		},
		{
			name: "Web app project name",
			projectInfo: map[string]interface{}{
				"project_name": "dashboard-webapp",
			},
			expectedType: DemoTypeWebApp,
			reasonCheck:  "web application",
		},
		{
			name: "CLI tool project name",
			projectInfo: map[string]interface{}{
				"project_name": "dev-cli-tool",
			},
			expectedType: DemoTypeCLITool,
			reasonCheck:  "CLI tool",
		},
		{
			name: "Data project name",
			projectInfo: map[string]interface{}{
				"project_name": "sales-data-analysis",
			},
			expectedType: DemoTypeDataAnalysis,
			reasonCheck:  "data analysis",
		},
		{
			name: "AI project name",
			projectInfo: map[string]interface{}{
				"project_name": "ml-recommender",
			},
			expectedType: DemoTypeAI,
			reasonCheck:  "AI/ML project",
		},
		{
			name: "React detected",
			projectInfo: map[string]interface{}{
				"project_name":  "my-project",
				"detected_tech": []string{"React", "Node.js"},
			},
			expectedType: DemoTypeWebApp,
			reasonCheck:  "React framework",
		},
		{
			name: "TensorFlow detected",
			projectInfo: map[string]interface{}{
				"project_name":  "my-project",
				"detected_tech": []string{"TensorFlow", "Python"},
			},
			expectedType: DemoTypeAI,
			reasonCheck:  "TensorFlow ML framework",
		},
		{
			name: "Pandas detected",
			projectInfo: map[string]interface{}{
				"project_name":  "my-project",
				"detected_tech": []string{"pandas", "jupyter"},
			},
			expectedType: DemoTypeDataAnalysis,
			reasonCheck:  "pandas data tool",
		},
		{
			name: "No hints - default",
			projectInfo: map[string]interface{}{
				"project_name": "generic-project",
			},
			expectedType: DemoTypeDefault,
			reasonCheck:  "No specific project type detected",
		},
		{
			name:         "Empty project info",
			projectInfo:  map[string]interface{}{},
			expectedType: DemoTypeDefault,
			reasonCheck:  "No specific project type detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			demoType, reason := generator.GetRecommendedDemo(ctx, tt.projectInfo)
			assert.Equal(t, tt.expectedType, demoType)
			assert.Contains(t, reason, tt.reasonCheck)
		})
	}
}

func TestGetRecommendedDemoWithCancellation(t *testing.T) {
	generator := NewDemoCommissionGenerator()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	demoType, reason := generator.GetRecommendedDemo(ctx, map[string]interface{}{})
	assert.Equal(t, DemoTypeDefault, demoType)
	assert.Contains(t, reason, "cancellation")
}

func TestDemoCommissionContent(t *testing.T) {
	generator := NewDemoCommissionGenerator()
	ctx := context.Background()

	// Test that each demo has substantial content
	types := generator.GetAvailableTypes()
	for _, demoType := range types {
		t.Run(string(demoType), func(t *testing.T) {
			content, err := generator.GenerateCommission(ctx, demoType)
			require.NoError(t, err)

			// Check content quality
			assert.Greater(t, len(content), 1000, "Demo should have substantial content")
			assert.Contains(t, content, "## ", "Should have section headers")
			assert.Contains(t, content, "Project Objective", "Should have objective")
			assert.Contains(t, content, "Technical", "Should have technical details")

			// Check for success/goals/metrics criteria (different demos use different terms)
			hasSuccessCriteria := strings.Contains(content, "Success") ||
				strings.Contains(content, "Goals") ||
				strings.Contains(content, "Metrics") ||
				strings.Contains(content, "Deliverables")
			assert.True(t, hasSuccessCriteria, "Should have success criteria, goals, metrics, or deliverables")

			assert.Contains(t, content, "Demonstration", "Should explain demonstration value")

			// Check for task lists
			assert.Contains(t, content, "- [ ]", "Should have task checkboxes")
		})
	}
}

func TestDemoCommissionTags(t *testing.T) {
	// Verify each template has appropriate tags
	templates := initializeDemoTemplates()

	for demoType, template := range templates {
		t.Run(string(demoType), func(t *testing.T) {
			assert.NotEmpty(t, template.Tags, "Demo should have tags")
			assert.Contains(t, template.Tags, "demo", "All demos should have 'demo' tag")

			// Type-specific tag checks
			switch demoType {
			case DemoTypeAPIService:
				assert.Contains(t, template.Tags, "api")
			case DemoTypeWebApp:
				assert.Contains(t, template.Tags, "webapp")
			case DemoTypeCLITool:
				assert.Contains(t, template.Tags, "cli")
			case DemoTypeDataAnalysis:
				assert.Contains(t, template.Tags, "data")
			case DemoTypeMicroservices:
				assert.Contains(t, template.Tags, "microservices")
			case DemoTypeAI:
				assert.Contains(t, template.Tags, "ai")
			}
		})
	}
}

func TestConcurrentCommissionGeneration(t *testing.T) {
	generator := NewDemoCommissionGenerator()
	ctx := context.Background()

	// Test concurrent access
	done := make(chan bool, 6)
	errors := make(chan error, 6)

	types := []DemoCommissionType{
		DemoTypeAPIService,
		DemoTypeWebApp,
		DemoTypeCLITool,
		DemoTypeDataAnalysis,
		DemoTypeMicroservices,
		DemoTypeAI,
	}

	for _, demoType := range types {
		go func(dt DemoCommissionType) {
			_, err := generator.GenerateCommission(ctx, dt)
			if err != nil {
				errors <- err
			}
			done <- true
		}(demoType)
	}

	// Wait for all goroutines with timeout
	timeout := time.After(5 * time.Second)
	for i := 0; i < 6; i++ {
		select {
		case <-done:
			// Success
		case err := <-errors:
			t.Errorf("Error during concurrent generation: %v", err)
		case <-timeout:
			t.Fatal("Timeout waiting for concurrent generation")
		}
	}
}
