package local

// LocalGuildStructure represents the structure of a local .guild directory
type LocalGuildStructure struct {
	// Core files
	ConfigPath   string // guild.yaml
	DatabasePath string // memory.db
	
	// Directories
	CommissionsDir  string // commissions/ - User objectives/goals
	CampaignsDir    string // campaigns/ - Execution plans
	KanbanDir       string // kanban/ - Task tracking
	CorpusDir       string // corpus/ - Project documentation
	PromptsDir      string // prompts/ - Custom templates
	ToolsDir        string // tools/ - Project-specific tool installations/configs
	WorkspacesDir   string // workspaces/ - Agent work areas
	// ArchivesDir  string // TODO: archives/ - Agent memory (pending ChromemGo deletion)
}

// ProjectInfo represents information about the current project
type ProjectInfo struct {
	Path        string
	Type        string // golang, python, typescript, rust, generic
	Name        string
	Description string
	VCSType     string // git, svn, none
	VCSRemote   string // remote URL if available
}

// LocalState represents the current state of a local Guild project
type LocalState struct {
	Initialized    bool
	HasDatabase    bool
	HasConfig      bool
	HasCommissions bool
	HasCorpus      bool
	ActiveAgents   int
	LastModified   string
}