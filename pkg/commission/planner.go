// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package commission

import (
	"context"

	"github.com/lancekrogers/guild-core/pkg/gerror"
)

// Planner manages the planning process for commissions
type Planner struct {
	manager          *Manager
	lifecycleManager *LifecycleManager
	session          *PlanningSession
}

// newPlanner creates a new commission planner (private constructor)
func newPlanner(manager *Manager, lifecycleManager *LifecycleManager) *Planner {
	return &Planner{
		manager:          manager,
		lifecycleManager: lifecycleManager,
		session:          newPlanningSession(),
	}
}

// DefaultPlannerFactory creates a planner factory for registry use
func DefaultPlannerFactory(manager *Manager, lifecycleManager *LifecycleManager) *Planner {
	return newPlanner(manager, lifecycleManager)
}

// GetSession returns the current planning session
func (p *Planner) GetSession() *PlanningSession {
	return p.session
}

// SetCommission sets the commission for planning
func (p *Planner) SetCommission(ctx context.Context, commissionID string) error {
	// Get the commission
	obj, err := p.manager.GetCommission(ctx, commissionID)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "commission").WithComponent("set_commission").WithOperation("failed to get commission")
	}

	// Create a new session with this commission
	p.session = NewPlanningSession()
	p.session.Commission = obj
	p.session.AddActivityLog("Commission loaded: " + obj.Title)

	return nil
}

// CreateCommission creates a new commission from a description
func (p *Planner) CreateCommission(ctx context.Context, description string) error {
	// Create commission via lifecycle manager
	obj, err := p.lifecycleManager.CreateCommissionFromDescription(ctx, description)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "commission").WithComponent("create_commission").WithOperation("failed to create commission")
	}

	// Set the commission in the session
	p.session = NewPlanningSession()
	p.session.Commission = obj
	p.session.AddActivityLog("Commission created: " + obj.Title)

	return nil
}

// AddContext adds context to the current commission
func (p *Planner) AddContext(ctx context.Context, contextText string) error {
	if p.session.Commission == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "no commission set in the planning session", nil).
			WithComponent("commission").
			WithOperation("add_context")
	}

	// Add context via lifecycle manager
	if err := p.lifecycleManager.AddContext(ctx, p.session.Commission.ID, contextText); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "commission").WithComponent("add_context").WithOperation("failed to add context")
	}

	// Refresh the commission
	obj, err := p.manager.GetCommission(ctx, p.session.Commission.ID)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "commission").WithComponent("refresh_commission").WithOperation("failed to refresh commission")
	}

	// Update the session
	p.session.Commission = obj
	p.session.AddActivityLog("Context added: " + truncateString(contextText, 50))
	p.session.ContextAdded = append(p.session.ContextAdded, contextText)

	return nil
}

// Regenerate regenerates documents for the current commission
func (p *Planner) Regenerate(ctx context.Context) error {
	if p.session.Commission == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "no commission set in the planning session", nil).
			WithComponent("commission").
			WithOperation("add_context")
	}

	// Generate project structure via lifecycle manager
	if err := p.lifecycleManager.GenerateProjectStructure(ctx, p.session.Commission.ID); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "commission").WithComponent("regenerate_commission").WithOperation("failed to regenerate documents")
	}

	// Refresh the commission
	obj, err := p.manager.GetCommission(ctx, p.session.Commission.ID)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "commission").WithComponent("refresh_commission").WithOperation("failed to refresh commission")
	}

	// Update the session
	p.session.Commission = obj
	p.session.AddActivityLog("Documents regenerated")
	p.session.RegenerationCount++

	return nil
}

// MarkReady marks the current commission as ready
func (p *Planner) MarkReady(ctx context.Context) error {
	if p.session.Commission == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "no commission set in the planning session", nil).
			WithComponent("commission").
			WithOperation("add_context")
	}

	// Mark as ready via lifecycle manager
	if err := p.lifecycleManager.MarkCommissionReady(ctx, p.session.Commission.ID); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "commission").WithComponent("finalize_commission").WithOperation("failed to mark commission as ready")
	}

	// Refresh the commission
	obj, err := p.manager.GetCommission(ctx, p.session.Commission.ID)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "commission").WithComponent("refresh_commission").WithOperation("failed to refresh commission")
	}

	// Update the session
	p.session.Commission = obj
	p.session.AddActivityLog("Commission marked as ready")
	p.session.IsReady = true

	return nil
}

// GetSuggestions gets improvement suggestions for the commission
func (p *Planner) GetSuggestions(ctx context.Context) (string, error) {
	if p.session.Commission == nil {
		return "", gerror.New(gerror.ErrCodeInvalidInput, "no commission set in the planning session", nil).
			WithComponent("commission").
			WithOperation("get_commission_status")
	}

	// In a real implementation, this would use an LLM to generate suggestions
	// For now, we'll return some static suggestions
	suggestions := `Based on the current commission, consider:

1. Add more specific requirements to clarify the expected outcomes
2. Include examples or use cases to illustrate your goal
3. Consider adding technical constraints or performance targets
4. Specify how this commission relates to other parts of the system
5. Tag the commission with relevant categories to improve organization`

	p.session.AddActivityLog("Suggestions requested and generated")
	p.session.Suggestions = suggestions

	return suggestions, nil
}

// CreateTaskPlan generates a task plan for an commission
func (p *Planner) CreateTaskPlan(ctx context.Context, commissionID string) ([]TaskPlan, error) {
	// Get the commission
	obj, err := p.manager.GetCommission(ctx, commissionID)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "commission").WithComponent("get_all_commissions").WithOperation("failed to get commission")
	}

	// In a real implementation, this would use an LLM to generate tasks
	// For now, we'll generate some example tasks based on the commission
	var tasks []TaskPlan

	// Add a planning task
	tasks = append(tasks, TaskPlan{
		Title:        "Plan implementation approach",
		Description:  "Determine the best approach to implement " + obj.Title,
		Priority:     "high",
		Dependencies: []string{},
	})

	// Add a research task
	tasks = append(tasks, TaskPlan{
		Title:        "Research existing solutions",
		Description:  "Research existing solutions and libraries for " + obj.Title,
		Priority:     "medium",
		Dependencies: []string{},
	})

	// Add an implementation task
	tasks = append(tasks, TaskPlan{
		Title:        "Implement core functionality",
		Description:  "Implement the core functionality for " + obj.Title,
		Priority:     "high",
		Dependencies: []string{"Plan implementation approach"},
	})

	// Add a testing task
	tasks = append(tasks, TaskPlan{
		Title:        "Write tests",
		Description:  "Write tests for " + obj.Title,
		Priority:     "medium",
		Dependencies: []string{"Implement core functionality"},
	})

	// Add a documentation task
	tasks = append(tasks, TaskPlan{
		Title:        "Document the implementation",
		Description:  "Document how to use the implementation of " + obj.Title,
		Priority:     "low",
		Dependencies: []string{"Implement core functionality"},
	})

	return tasks, nil
}

// TaskPlan represents a planned task for an commission
type TaskPlan struct {
	Title        string
	Description  string
	Priority     string
	Dependencies []string
}

// Helper function to truncate a string
func truncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength-3] + "..."
}
