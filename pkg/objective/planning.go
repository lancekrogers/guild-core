package objective

import (
	"time"
)

// PlanningSession represents a planning session for an objective
type PlanningSession struct {
	// ObjectiveID is the ID of the objective being planned
	ObjectiveID string
	
	// StartTime is when the session started
	StartTime time.Time
	
	// Documents is a map of document reference IDs to their content
	Documents map[string]string
	
	// Notes is a list of notes added during the session
	Notes []string
	
	// Tasks is a list of tasks created during the session
	Tasks []*TaskPlan
}

// NewPlanningSession creates a new planning session
func NewPlanningSession(objectiveID string) *PlanningSession {
	return &PlanningSession{
		ObjectiveID: objectiveID,
		StartTime:   time.Now().UTC(),
		Documents:   make(map[string]string),
		Notes:       make([]string, 0),
		Tasks:       make([]*TaskPlan, 0),
	}
}

// AddNote adds a note to the session
func (ps *PlanningSession) AddNote(note string) {
	ps.Notes = append(ps.Notes, note)
}

// AddTask adds a task to the session
func (ps *PlanningSession) AddTask(task *TaskPlan) {
	ps.Tasks = append(ps.Tasks, task)
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