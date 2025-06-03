package boltdb

// Bucket names used by the BoltDB store
const (
	// BucketPromptChains stores prompt chains by ID
	BucketPromptChains = "prompt_chains"
	
	// BucketPromptChainsByAgent indexes chains by agent ID
	BucketPromptChainsByAgent = "prompt_chains_by_agent"
	
	// BucketPromptChainsByTask indexes chains by task ID
	BucketPromptChainsByTask = "prompt_chains_by_task"
	
	// BucketCorpusDocuments stores corpus documents
	BucketCorpusDocuments = "corpus_documents"
	
	// BucketCorpusMetadata stores document metadata
	BucketCorpusMetadata = "corpus_metadata"
	
	// BucketTasks stores kanban tasks
	BucketTasks = "tasks"
	
	// BucketTasksByStatus indexes tasks by status
	BucketTasksByStatus = "tasks_by_status"
	
	// BucketTasksByAgent indexes tasks by agent
	BucketTasksByAgent = "tasks_by_agent"
	
	// BucketConfig stores system configuration
	BucketConfig = "config"
	
	// BucketBoards stores kanban boards
	BucketBoards = "boards"
)

// AllBuckets returns all bucket names
// Note: The "objectives" bucket is not included here as it's managed
// separately by the objective package. Use WithCustomBuckets("objectives")
// when creating a store for the objective manager.
func AllBuckets() []string {
	return []string{
		BucketPromptChains,
		BucketPromptChainsByAgent,
		BucketPromptChainsByTask,
		BucketCorpusDocuments,
		BucketCorpusMetadata,
		BucketTasks,
		BucketTasksByStatus,
		BucketTasksByAgent,
		BucketConfig,
		BucketBoards,
	}
}