// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package workflows

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lancekrogers/guild/internal/ui/chat/components"
	"github.com/lancekrogers/guild/pkg/agents/core/elena"
	"github.com/lancekrogers/guild/pkg/commission"
	"github.com/lancekrogers/guild/pkg/gerror"
)

// CommissionWorkflow manages the commission creation and refinement workflow
type CommissionWorkflow struct {
	ctx        context.Context
	dialogue   *elena.PlanningDialogue
	generator  *commission.Generator
	manager    commission.CommissionManager
	commission *commission.Commission
	state      WorkflowState
	editMode   bool
	formatter  *components.CommissionFormatter
}

// WorkflowState represents the current state of the commission workflow
type WorkflowState string

const (
	StateDialogue    WorkflowState = "dialogue"
	StateDraftReview WorkflowState = "draft_review"
	StateRefinement  WorkflowState = "refinement"
	StateComplete    WorkflowState = "complete"
)

// NewCommissionWorkflow creates a new commission workflow
func NewCommissionWorkflow(ctx context.Context, generator *commission.Generator, manager commission.CommissionManager) *CommissionWorkflow {
	// Create formatter with reasonable default width
	formatter, err := components.NewCommissionFormatter(ctx, 80)
	if err != nil {
		// Fallback to nil formatter if creation fails
		formatter = nil
	}

	return &CommissionWorkflow{
		ctx:       ctx,
		generator: generator,
		manager:   manager,
		state:     StateDialogue,
		dialogue:  elena.NewPlanningDialogue(fmt.Sprintf("dialogue_%d", time.Now().Unix())),
		formatter: formatter,
	}
}

// Start begins the commission workflow
func (cw *CommissionWorkflow) Start() tea.Cmd {
	return tea.Batch(
		func() tea.Msg {
			return CommissionProgressMsg{
				Stage:    StageIntroduction,
				Progress: 0.0,
				Status:   "Starting commission planning",
			}
		},
		func() tea.Msg {
			return CommissionWorkflowMsg{
				Type:    "start",
				Content: cw.dialogue.GetNextQuestion(),
			}
		},
	)
}

// ProcessInput processes user input during the workflow
func (cw *CommissionWorkflow) ProcessInput(input string) tea.Cmd {
	switch cw.state {
	case StateDialogue:
		return cw.processDialogueInput(input)
	case StateDraftReview:
		return cw.processDraftReviewInput(input)
	case StateRefinement:
		return cw.processRefinementInput(input)
	default:
		return nil
	}
}

// processDialogueInput handles input during the dialogue phase
func (cw *CommissionWorkflow) processDialogueInput(input string) tea.Cmd {
	// Process the response in the dialogue
	if err := cw.dialogue.ProcessResponse(cw.ctx, input); err != nil {
		return func() tea.Msg {
			return CommissionWorkflowMsg{
				Type:  "error",
				Error: err,
			}
		}
	}

	// Check if dialogue is complete
	if cw.dialogue.IsComplete() {
		// Generate commission from dialogue
		return tea.Batch(
			func() tea.Msg {
				return CommissionProgressMsg{
					Stage:    StageSummary,
					Progress: 1.0,
					Status:   "Generating commission document",
				}
			},
			cw.generateCommission(),
		)
	}

	// Calculate progress and stage based on dialogue state
	stage, progress := cw.calculateProgress()

	// Get next question with progress update
	return tea.Batch(
		func() tea.Msg {
			return CommissionProgressMsg{
				Stage:    stage,
				Progress: progress,
				Status:   fmt.Sprintf("Gathering %s information", stage.String()),
			}
		},
		func() tea.Msg {
			return CommissionWorkflowMsg{
				Type:    "question",
				Content: cw.dialogue.GetNextQuestion(),
			}
		},
	)
}

// generateCommission generates a commission from the completed dialogue
func (cw *CommissionWorkflow) generateCommission() tea.Cmd {
	return func() tea.Msg {
		// Generate the commission
		commission, err := cw.generator.GenerateFromDialogue(cw.ctx, cw.dialogue)
		if err != nil {
			return CommissionWorkflowMsg{
				Type:  "error",
				Error: err,
			}
		}

		cw.commission = commission
		cw.state = StateDraftReview

		// Return draft for review
		return CommissionWorkflowMsg{
			Type:    "draft",
			Content: cw.formatCommissionForDisplay(commission),
			Data:    commission,
		}
	}
}

// processDraftReviewInput handles input during draft review
func (cw *CommissionWorkflow) processDraftReviewInput(input string) tea.Cmd {
	lower := strings.ToLower(strings.TrimSpace(input))

	// Check various approval phrases
	if lower == "yes" || lower == "y" ||
		strings.Contains(lower, "looks good") ||
		strings.Contains(lower, "proceed") ||
		strings.Contains(lower, "save") ||
		lower == "1" {
		// Save the commission
		return cw.saveCommission()
	}

	// Check for edit requests
	if strings.Contains(lower, "edit") || strings.Contains(lower, "change") || strings.Contains(lower, "modify") {
		cw.editMode = true
		return func() tea.Msg {
			return CommissionWorkflowMsg{
				Type: "edit_prompt",
				Content: `What would you like to modify?

You can say things like:
- "Change the requirements section"
- "Add more technology details"
- "Update the timeline"
- "Fix the project description"

Or type specific content to add or change.`,
			}
		}
	}

	// Check for cancel
	if strings.Contains(lower, "cancel") || strings.Contains(lower, "abort") {
		return func() tea.Msg {
			return CommissionWorkflowMsg{
				Type:    "cancelled",
				Content: "Commission creation cancelled. You can start again with '/commission new'.",
			}
		}
	}

	// Unclear input
	return func() tea.Msg {
		return CommissionWorkflowMsg{
			Type: "clarification",
			Content: `I didn't understand your response. Would you like to:

1. **Save** this commission as is
2. **Edit** specific sections
3. **Cancel** and start over

Please type your choice or describe what you'd like to change.`,
		}
	}
}

// saveCommission saves the commission to disk
func (cw *CommissionWorkflow) saveCommission() tea.Cmd {
	return func() tea.Msg {
		// Save via manager
		if err := cw.manager.SaveCommission(cw.ctx, cw.commission); err != nil {
			return CommissionWorkflowMsg{
				Type: "error",
				Error: gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save commission").
					WithComponent("commission.workflow").
					WithOperation("saveCommission"),
			}
		}

		cw.state = StateComplete

		return CommissionWorkflowMsg{
			Type: "saved",
			Content: fmt.Sprintf(`✅ Commission saved successfully!

**ID**: %s
**Title**: %s
**Location**: commissions/%s.md

You can now:
- Trigger refinement with '/commission refine %s'
- View the commission in your project's commissions directory
- Continue with other tasks

The Guild stands ready to execute this commission!`,
				cw.commission.ID,
				cw.commission.Title,
				cw.commission.ID,
				cw.commission.ID),
			Data: cw.commission,
		}
	}
}

// formatCommissionForDisplay formats a commission for chat display using enhanced formatting
func (cw *CommissionWorkflow) formatCommissionForDisplay(comm *commission.Commission) string {
	// Use enhanced formatter if available
	if cw.formatter != nil {
		formatted, err := cw.formatter.FormatDraft(comm)
		if err == nil {
			return formatted
		}
		// Fall back to basic formatting if enhanced formatting fails
	}

	// Fallback to basic formatting
	return cw.formatCommissionBasic(comm)
}

// formatCommissionBasic provides basic commission formatting as fallback
func (cw *CommissionWorkflow) formatCommissionBasic(comm *commission.Commission) string {
	var sb strings.Builder

	// Header
	sb.WriteString("📜 **Commission Draft**\n")
	sb.WriteString("════════════════════\n\n")

	// Use the commission's Format method to get markdown
	sb.WriteString(comm.Format())

	// Add the generated content if available
	if comm.Content != "" {
		sb.WriteString("\n---\n\n")
		sb.WriteString("**Full Document:**\n\n")
		sb.WriteString(comm.Content)
	}

	// Footer with options
	sb.WriteString("\n\n---\n\n")
	sb.WriteString("**Review Options:**\n\n")
	sb.WriteString("• Type **'yes'** or **'save'** to save this commission\n")
	sb.WriteString("• Type **'edit'** to modify sections\n")
	sb.WriteString("• Type **'cancel'** to abort\n")

	return sb.String()
}

// processRefinementInput handles input during refinement
func (cw *CommissionWorkflow) processRefinementInput(input string) tea.Cmd {
	// This would handle the refinement workflow
	// For now, just acknowledge
	return func() tea.Msg {
		return CommissionWorkflowMsg{
			Type:    "info",
			Content: "Refinement workflow not yet implemented. Commission saved as draft.",
		}
	}
}

// GetState returns the current workflow state
func (cw *CommissionWorkflow) GetState() WorkflowState {
	return cw.state
}

// IsComplete returns true if the workflow is complete
func (cw *CommissionWorkflow) IsComplete() bool {
	return cw.state == StateComplete
}

// calculateProgress calculates the current stage and progress based on dialogue state
func (cw *CommissionWorkflow) calculateProgress() (PlanningStage, float64) {
	if cw.dialogue == nil {
		return StageIntroduction, 0.0
	}

	// Use response count to estimate progress since dialogue doesn't expose current stage
	responseCount := len(cw.dialogue.GetResponses())
	var stage PlanningStage
	var progress float64

	if responseCount == 0 {
		stage = StageIntroduction
		progress = 0.0
	} else if responseCount <= 2 {
		stage = StageProjectType
		progress = 0.25
	} else if responseCount <= 4 {
		stage = StageRequirements
		progress = 0.5
	} else if responseCount <= 6 {
		stage = StageTechnology
		progress = 0.75
	} else {
		stage = StageConstraints
		progress = 0.9
	}

	return stage, progress
}

// CommissionWorkflowMsg represents messages from the commission workflow
type CommissionWorkflowMsg struct {
	Type    string
	Content string
	Data    interface{}
	Error   error
}

// CommissionProgressMsg represents commission progress updates
type CommissionProgressMsg struct {
	Stage    PlanningStage
	Progress float64
	Status   string
}

// PlanningStage represents commission planning stages (imported from panes)
type PlanningStage int

const (
	StageIntroduction PlanningStage = iota
	StageProjectType
	StageRequirements
	StageTechnology
	StageConstraints
	StageSummary
)

// String returns a human-readable stage name
func (ps PlanningStage) String() string {
	stages := []string{
		"Introduction",
		"Project Type",
		"Requirements",
		"Technology",
		"Constraints",
		"Summary",
	}
	if int(ps) < len(stages) {
		return stages[ps]
	}
	return "Unknown"
}

// Styling helpers

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))

	sectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39"))

	optionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)

// FormatCommissionSection formats a section with consistent styling
func FormatCommissionSection(title, content string) string {
	return fmt.Sprintf("%s\n%s\n",
		titleStyle.Render(title),
		content)
}
