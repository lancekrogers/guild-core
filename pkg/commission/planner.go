package commission

import (
	"context"

	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// Planner manages the planning process for objectives
type Planner struct {
	manager         *Manager
	lifecycleManager *LifecycleManager
	session         *PlanningSession
}

// newPlanner creates a new objective planner (private constructor)
func newPlanner(manager *Manager, lifecycleManager *LifecycleManager) *Planner {
	return &Planner{
		manager:         manager,
		lifecycleManager: lifecycleManager,
		session:         newPlanningSession(),
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
		return gerror.Wrap(err, gerror.Internal, "commission", "set_commission", "failed to get commission")
	}

	// Create a new session with this commission
	p.session = NewPlanningSession()
	p.session.Commission = obj
	p.session.AddActivityLog("Commission loaded: " + obj.Title)

	return nil
}

// CreateObjective creates a new objective from a description
func (p *Planner) CreateObjective(ctx context.Context, description string) error {
	// Create objective via lifecycle manager
	obj, err := p.lifecycleManager.CreateObjectiveFromDescription(ctx, description)
	if err != nil {
		return gerror.Wrap(err, gerror.Internal, "commission", "create_objective", "failed to create objective")
	}

	// Set the objective in the session
	p.session = NewPlanningSession()
	p.session.Commission = obj
	p.session.AddActivityLog("Objective created: " + obj.Title)

	return nil
}

// AddContext adds context to the current objective
func (p *Planner) AddContext(ctx context.Context, contextText string) error {
	if p.session.Commission == nil {
		return gerror.New(gerror.InvalidArgument, "commission", "add_context", "no objective set in the planning session")
	}

	// Add context via lifecycle manager
	if err := p.lifecycleManager.AddContext(ctx, p.session.Commission.ID, contextText); err != nil {
		return gerror.Wrap(err, gerror.Internal, "commission", "add_context", "failed to add context")
	}

	// Refresh the objective
	obj, err := p.manager.GetObjective(ctx, p.session.Commission.ID)
	if err != nil {
		return gerror.Wrap(err, gerror.Internal, "commission", "refresh_objective", "failed to refresh objective")
	}

	// Update the session
	p.session.Commission = obj
	p.session.AddActivityLog("Context added: " + truncateString(contextText, 50))
	p.session.ContextAdded = append(p.session.ContextAdded, contextText)

	return nil
}

// Regenerate regenerates documents for the current objective
func (p *Planner) Regenerate(ctx context.Context) error {
	if p.session.Commission == nil {
		return gerror.New(gerror.InvalidArgument, "commission", "add_context", "no objective set in the planning session")
	}

	// Generate project structure via lifecycle manager
	if err := p.lifecycleManager.GenerateProjectStructure(ctx, p.session.Commission.ID); err != nil {
		return gerror.Wrap(err, gerror.Internal, "commission", "regenerate_objective", "failed to regenerate documents")
	}

	// Refresh the objective
	obj, err := p.manager.GetObjective(ctx, p.session.Commission.ID)
	if err != nil {
		return gerror.Wrap(err, gerror.Internal, "commission", "refresh_objective", "failed to refresh objective")
	}

	// Update the session
	p.session.Commission = obj
	p.session.AddActivityLog("Documents regenerated")
	p.session.RegenerationCount++

	return nil
}

// MarkReady marks the current objective as ready
func (p *Planner) MarkReady(ctx context.Context) error {
	if p.session.Commission == nil {
		return gerror.New(gerror.InvalidArgument, "commission", "add_context", "no objective set in the planning session")
	}

	// Mark as ready via lifecycle manager
	if err := p.lifecycleManager.MarkObjectiveReady(ctx, p.session.Commission.ID); err != nil {
		return gerror.Wrap(err, gerror.Internal, "commission", "finalize_objective", "failed to mark objective as ready")
	}

	// Refresh the objective
	obj, err := p.manager.GetObjective(ctx, p.session.Commission.ID)
	if err != nil {
		return gerror.Wrap(err, gerror.Internal, "commission", "refresh_objective", "failed to refresh objective")
	}

	// Update the session
	p.session.Commission = obj
	p.session.AddActivityLog("Objective marked as ready")
	p.session.IsReady = true

	return nil
}

// GetSuggestions gets improvement suggestions for the objective
func (p *Planner) GetSuggestions(ctx context.Context) (string, error) {
	if p.session.Commission == nil {
		return "", gerror.New(gerror.InvalidArgument, "commission", "get_objective_status", "no objective set in the planning session")
	}

	// In a real implementation, this would use an LLM to generate suggestions
	// For now, we'll return some static suggestions
	suggestions := `Based on the current objective, consider:

1. Add more specific requirements to clarify the expected outcomes
2. Include examples or use cases to illustrate your goal
3. Consider adding technical constraints or performance targets
4. Specify how this objective relates to other parts of the system
5. Tag the objective with relevant categories to improve organization`

	p.session.AddActivityLog("Suggestions requested and generated")
	p.session.Suggestions = suggestions

	return suggestions, nil
}

// CreateTaskPlan generates a task plan for an objective
func (p *Planner) CreateTaskPlan(ctx context.Context, objectiveID string) ([]TaskPlan, error) {
	// Get the objective
	obj, err := p.manager.GetObjective(ctx, objectiveID)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.Internal, "commission", "get_all_objectives", "failed to get objective")
	}

	// In a real implementation, this would use an LLM to generate tasks
	// For now, we'll generate some example tasks based on the objective
	var tasks []TaskPlan

	// Add a planning task
	tasks = append(tasks, TaskPlan{
		Title:       "Plan implementation approach",
		Description: "Determine the best approach to implement " + obj.Title,
		Priority:    "high",
		Dependencies: []string{},
	})

	// Add a research task
	tasks = append(tasks, TaskPlan{
		Title:       "Research existing solutions",
		Description: "Research existing solutions and libraries for " + obj.Title,
		Priority:    "medium",
		Dependencies: []string{},
	})

	// Add an implementation task
	tasks = append(tasks, TaskPlan{
		Title:       "Implement core functionality",
		Description: "Implement the core functionality for " + obj.Title,
		Priority:    "high",
		Dependencies: []string{"Plan implementation approach"},
	})

	// Add a testing task
	tasks = append(tasks, TaskPlan{
		Title:       "Write tests",
		Description: "Write tests for " + obj.Title,
		Priority:    "medium",
		Dependencies: []string{"Implement core functionality"},
	})

	// Add a documentation task
	tasks = append(tasks, TaskPlan{
		Title:       "Document the implementation",
		Description: "Document how to use the implementation of " + obj.Title,
		Priority:    "low",
		Dependencies: []string{"Implement core functionality"},
	})

	return tasks, nil
}

// TaskPlan represents a planned task for an objective
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