package execution

import "time"

// ExecutionPromptData contains all data needed to build execution prompts
type ExecutionPromptData struct {
	Agent      AgentData
	Context    ContextData
	Commission CommissionData
	Task       TaskData
	Tools      []ToolData
	ToolConfig ToolConfigData
	Execution  ExecutionData
}

// AgentData contains agent-specific information
type AgentData struct {
	Name         string
	Role         string
	Capabilities []string
}

// ContextData contains project and guild context
type ContextData struct {
	GuildID            string
	ProjectName        string
	ProjectDescription string
	WorkspaceDir       string
	RelevantDocs       []DocumentRef
	TechStack          string
	Architecture       string
	Dependencies       string
	RelatedTasks       []RelatedTask
}

// DocumentRef represents a reference to relevant documentation
type DocumentRef struct {
	Path    string
	Summary string
}

// RelatedTask represents a task related to the current one
type RelatedTask struct {
	Title      string
	Status     string
	AssignedTo string
	Output     string
}

// CommissionData contains commission (objective) information
type CommissionData struct {
	Title            string
	Description      string
	SuccessCriteria  []string
}

// TaskData contains current task information
type TaskData struct {
	Title          string
	Description    string
	Requirements   []string
	Constraints    []string
	Priority       string
	DueDate        string
	EstimatedHours float64
	Dependencies   []TaskDependency
	Deliverables   []Deliverable
}

// TaskDependency represents a task this one depends on
type TaskDependency struct {
	TaskID     string
	Title      string
	Status     string
	OutputPath string
}

// Deliverable represents an expected output
type Deliverable struct {
	Name         string
	Type         string
	Format       string
	ExpectedPath string
	Description  string
}

// ToolData contains tool (implement) information
type ToolData struct {
	Name        string
	Description string
	Usage       string
	Parameters  []ToolParameter
	ReturnType  string
	Example     string
}

// ToolParameter represents a tool parameter
type ToolParameter struct {
	Name        string
	Type        string
	Description string
}

// ToolConfigData contains tool usage configuration
type ToolConfigData struct {
	MaxCalls   int
	Timeout    time.Duration
	RateLimits string
}

// ExecutionData contains current execution state
type ExecutionData struct {
	Phase                  string
	StepNumber             int
	TotalSteps             int
	StepName               string
	StepObjective          string
	ExpectedActions        []string
	SuccessIndicators      []string
	PotentialIssues        []string
	OverallProgress        int
	PhaseProgress          int
	TimeElapsed            string
	EstimatedTimeRemaining string
	PreviousStepResult     string
	NextSteps              []string
}