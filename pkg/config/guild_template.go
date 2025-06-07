package config

// DefaultGuildTemplate returns a default guild configuration template
func DefaultGuildTemplate() *GuildConfig {
	return &GuildConfig{
		Name:        "MyGuild",
		Description: "A guild of AI agents working together",
		Version:     "1.0.0",
		Manager: ManagerConfig{
			Default: "orchestrator",
			Fallback: []string{"analyst", "coder"},
		},
		Agents: []AgentConfig{
			{
				ID:          "orchestrator",
				Name:        "Master Orchestrator",
				Type:        "manager",
				Provider:    "anthropic",
				Model:       "claude-3-opus-20240229",
				Description: "Primary manager agent for planning and task decomposition",
				Capabilities: []string{
					"planning",
					"task_decomposition",
					"architecture",
					"coordination",
					"analysis",
				},
				MaxTokens:   4096,
				Temperature: 0.7,
			},
			{
				ID:          "coder",
				Name:        "Code Artisan",
				Type:        "worker",
				Provider:    "openai",
				Model:       "gpt-4-turbo-preview",
				Description: "Specialist in coding, debugging, and technical implementation",
				Capabilities: []string{
					"coding",
					"debugging",
					"testing",
					"refactoring",
					"code_review",
				},
				Tools: []string{
					"file_edit",
					"shell_execute",
					"file_read",
				},
				MaxTokens:   4096,
				Temperature: 0.3,
			},
			{
				ID:          "analyst",
				Name:        "Research Analyst",
				Type:        "worker",
				Provider:    "anthropic",
				Model:       "claude-3-sonnet-20240229",
				Description: "Expert in research, analysis, and documentation",
				Capabilities: []string{
					"research",
					"analysis",
					"documentation",
					"summarization",
					"data_analysis",
				},
				Tools: []string{
					"web_search",
					"corpus_query",
					"file_read",
				},
				MaxTokens:   4096,
				Temperature: 0.5,
			},
			{
				ID:          "reviewer",
				Name:        "Quality Guardian",
				Type:        "specialist",
				Provider:    "openai",
				Model:       "gpt-4",
				Description: "Specialist in quality assurance, testing, and review",
				Capabilities: []string{
					"code_review",
					"testing",
					"quality_assurance",
					"security_review",
					"documentation_review",
				},
				Tools: []string{
					"file_read",
					"shell_execute",
				},
				MaxTokens:   2048,
				Temperature: 0.2,
			},
		},
		Metadata: Metadata{
			Tags: []string{"default", "template"},
		},
	}
}

// MinimalGuildTemplate returns a minimal guild configuration for simple projects
func MinimalGuildTemplate() *GuildConfig {
	return &GuildConfig{
		Name:        "SimpleGuild",
		Description: "A minimal guild configuration",
		Version:     "1.0.0",
		Manager: ManagerConfig{
			Default: "assistant",
		},
		Agents: []AgentConfig{
			{
				ID:       "assistant",
				Name:     "General Assistant",
				Type:     "manager",
				Provider: "openai",
				Model:    "gpt-4",
				Capabilities: []string{
					"planning",
					"coding",
					"analysis",
					"documentation",
				},
				Tools: []string{
					"file_edit",
					"file_read",
					"shell_execute",
				},
				MaxTokens:   4096,
				Temperature: 0.7,
			},
		},
	}
}

// ExampleGuildTemplates returns a map of example templates
func ExampleGuildTemplates() map[string]*GuildConfig {
	return map[string]*GuildConfig{
		"default": DefaultGuildTemplate(),
		"minimal": MinimalGuildTemplate(),
		"web_dev": &GuildConfig{
			Name:        "WebDevGuild",
			Description: "Guild specialized for web development projects",
			Manager: ManagerConfig{
				Default: "tech_lead",
			},
			Agents: []AgentConfig{
				{
					ID:          "tech_lead",
					Name:        "Technical Lead",
					Type:        "manager",
					Provider:    "anthropic",
					Model:       "claude-3-opus-20240229",
					Capabilities: []string{
						"planning",
						"architecture",
						"tech_stack_selection",
						"api_design",
					},
				},
				{
					ID:          "frontend_dev",
					Name:        "Frontend Developer",
					Type:        "worker",
					Provider:    "openai",
					Model:       "gpt-4-turbo-preview",
					Capabilities: []string{
						"frontend",
						"react",
						"css",
						"javascript",
						"ui_ux",
					},
					Tools: []string{
						"file_edit",
						"shell_execute",
					},
				},
				{
					ID:          "backend_dev",
					Name:        "Backend Developer",
					Type:        "worker",
					Provider:    "anthropic",
					Model:       "claude-3-sonnet-20240229",
					Capabilities: []string{
						"backend",
						"api_development",
						"database",
						"golang",
						"python",
					},
					Tools: []string{
						"file_edit",
						"shell_execute",
						"http_request",
					},
				},
			},
		},
		"data_science": &GuildConfig{
			Name:        "DataScienceGuild",
			Description: "Guild specialized for data science and ML projects",
			Manager: ManagerConfig{
				Default: "data_architect",
			},
			Agents: []AgentConfig{
				{
					ID:          "data_architect",
					Name:        "Data Architect",
					Type:        "manager",
					Provider:    "anthropic",
					Model:       "claude-3-opus-20240229",
					Capabilities: []string{
						"planning",
						"data_architecture",
						"ml_design",
						"pipeline_design",
					},
				},
				{
					ID:          "data_engineer",
					Name:        "Data Engineer",
					Type:        "worker",
					Provider:    "openai",
					Model:       "gpt-4",
					Capabilities: []string{
						"data_engineering",
						"etl",
						"sql",
						"python",
						"spark",
					},
					Tools: []string{
						"file_edit",
						"shell_execute",
						"sql_query",
					},
				},
				{
					ID:          "ml_engineer",
					Name:        "ML Engineer",
					Type:        "worker",
					Provider:    "anthropic",
					Model:       "claude-3-sonnet-20240229",
					Capabilities: []string{
						"machine_learning",
						"deep_learning",
						"model_training",
						"python",
						"tensorflow",
						"pytorch",
					},
					Tools: []string{
						"file_edit",
						"shell_execute",
						"jupyter_notebook",
					},
				},
			},
		},
	}
}
