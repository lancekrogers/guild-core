package commission

import (
	"time"
)

// PlanningSession represents a planning session for an objective
type PlanningSession struct {
	// Commission is the commission being planned
	Commission *Commission

	// StartTime is when the session started
	StartTime time.Time

	// Documents is a map of document reference IDs to their content
	Documents map[string]string

	// ActivityLog tracks all activity in the session
	ActivityLog []ActivityEntry

	// ContextAdded is the context added during this session
	ContextAdded []string

	// RegenerationCount is the number of times documents have been regenerated
	RegenerationCount int

	// IsReady indicates if the objective is marked as ready
	IsReady bool

	// Suggestions contains improvement suggestions
	Suggestions string

	// Tasks is a list of tasks created during the session
	Tasks []*TaskPlan
}

// ActivityEntry represents an entry in the activity log
type ActivityEntry struct {
	Timestamp time.Time
	Message   string
}

// newPlanningSession creates a new planning session (private constructor)
func newPlanningSession() *PlanningSession {
	return &PlanningSession{
		StartTime:         time.Now().UTC(),
		Documents:         make(map[string]string),
		ActivityLog:       make([]ActivityEntry, 0),
		ContextAdded:      make([]string, 0),
		RegenerationCount: 0,
		IsReady:           false,
		Tasks:             make([]*TaskPlan, 0),
	}
}

// DefaultPlanningSessionFactory creates a planning session factory for registry use
func DefaultPlanningSessionFactory() *PlanningSession {
	return newPlanningSession()
}

// AddActivityLog adds an entry to the activity log
func (ps *PlanningSession) AddActivityLog(message string) {
	entry := ActivityEntry{
		Timestamp: time.Now().UTC(),
		Message:   message,
	}
	ps.ActivityLog = append(ps.ActivityLog, entry)
}

// AddTask adds a task to the session
func (ps *PlanningSession) AddTask(task *TaskPlan) {
	ps.Tasks = append(ps.Tasks, task)
	ps.AddActivityLog("Task added: " + task.Title)
}

// GetDocumentCount returns the number of documents in the session
func (ps *PlanningSession) GetDocumentCount() int {
	return len(ps.Documents)
}

// GetDocumentTitles returns a list of document reference titles
func (ps *PlanningSession) GetDocumentTitles() []string {
	titles := make([]string, 0, len(ps.Documents))
	for title := range ps.Documents {
		titles = append(titles, title)
	}
	return titles
}

// GetSessionSummary returns a summary of the planning session
func (ps *PlanningSession) GetSessionSummary() map[string]interface{} {
	summary := make(map[string]interface{})

	// Basic session info
	summary["start_time"] = ps.StartTime.Format(time.RFC3339)
	summary["duration"] = time.Since(ps.StartTime).String()

	// Objective info
	if ps.Commission != nil {
		summary["objective_id"] = ps.Commission.ID
		summary["objective_title"] = ps.Commission.Title
		summary["objective_status"] = ps.Commission.Status
		summary["objective_completion"] = ps.Commission.Completion
	}

	// Activity stats
	summary["context_added_count"] = len(ps.ContextAdded)
	summary["regeneration_count"] = ps.RegenerationCount
	summary["activity_count"] = len(ps.ActivityLog)
	summary["task_count"] = len(ps.Tasks)
	summary["is_ready"] = ps.IsReady

	return summary
}

// GetActivityLog returns the complete activity log
func (ps *PlanningSession) GetActivityLog() []string {
	log := make([]string, len(ps.ActivityLog))
	for i, entry := range ps.ActivityLog {
		log[i] = entry.Timestamp.Format("15:04:05") + " - " + entry.Message
	}
	return log
}