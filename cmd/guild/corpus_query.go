package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/guild-ventures/guild-core/internal/corpus"
	corpusagent "github.com/guild-ventures/guild-core/internal/corpus/agent"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/memory/rag"
	"github.com/guild-ventures/guild-core/pkg/memory/vector"
	"github.com/guild-ventures/guild-core/pkg/providers"
	"github.com/guild-ventures/guild-core/pkg/providers/interfaces"
)

// corpusQueryCmd represents the corpus query command
var corpusQueryCmd = &cobra.Command{
	Use:   "query [question]",
	Short: "Query the corpus using the Corpus Agent",
	Long: `Query the corpus knowledge base using natural language questions.
	
The Corpus Agent searches through all stored knowledge (both human-curated 
and agent-generated) in the RAG system and synthesizes comprehensive answers.

You can ask questions like:
- "What are the main components of the Guild Framework?"
- "How does the agent orchestration system work?"
- "Explain the corpus and RAG integration"

The agent will search relevant documents and generate a synthesized response.`,
	Run: runCorpusQuery,
}

// corpusChatCmd represents the corpus chat command
var corpusChatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Interactive chat with the Corpus Agent",
	Long: `Start an interactive chat session with the Corpus Agent.
	
This allows you to have a conversation with the agent, asking follow-up 
questions and refining your queries. The agent maintains conversation 
context for more coherent responses.

Commands:
- Type your question and press Enter
- Type 'save' to save the last response as a corpus document
- Type 'clear' to clear conversation history
- Type 'exit' or 'quit' to end the session`,
	Run: runCorpusChat,
}

func runCorpusQuery(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	
	// Get query from args or flag
	var query string
	if len(args) > 0 {
		query = strings.Join(args, " ")
	} else {
		fmt.Println("Please provide a question")
		return
	}
	
	// Get flags
	saveDoc, _ := cmd.Flags().GetBool("save")
	title, _ := cmd.Flags().GetString("title")
	providerType, _ := cmd.Flags().GetString("provider")
	model, _ := cmd.Flags().GetString("model")
	
	// Initialize Corpus Agent
	corpusAgent, err := initializeCorpusAgent(providerType, model)
	if err != nil {
		fmt.Printf("Error initializing Corpus Agent: %v\n", err)
		return
	}
	
	// Execute query
	fmt.Println("Searching knowledge base...")
	response, err := corpusAgent.Execute(ctx, query)
	if err != nil {
		fmt.Printf("Error querying corpus: %v\n", err)
		return
	}
	
	// Display response
	fmt.Println("\n=== Response ===")
	fmt.Println(response)
	
	// Optionally save as document
	if saveDoc {
		if title == "" {
			// Generate title from query
			title = generateTitle(query)
		}
		
		doc, err := corpusAgent.GenerateDocument(ctx, query, title)
		if err != nil {
			fmt.Printf("\nError generating document: %v\n", err)
			return
		}
		
		if err := corpusAgent.SaveGeneratedDocument(ctx, doc); err != nil {
			fmt.Printf("\nError saving document: %v\n", err)
			return
		}
		
		fmt.Printf("\nDocument saved as: %s\n", doc.FilePath)
	}
}

func runCorpusChat(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	
	// Get flags
	providerType, _ := cmd.Flags().GetString("provider")
	model, _ := cmd.Flags().GetString("model")
	
	// Initialize Corpus Agent
	corpusAgent, err := initializeCorpusAgent(providerType, model)
	if err != nil {
		fmt.Printf("Error initializing Corpus Agent: %v\n", err)
		return
	}
	
	// Start chat interface
	fmt.Println("=== Corpus Agent Chat ===")
	fmt.Println("Ask questions about the knowledge base. Type 'help' for commands.")
	fmt.Println()
	
	scanner := bufio.NewScanner(os.Stdin)
	var lastResponse string
	
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		
		input := strings.TrimSpace(scanner.Text())
		
		// Handle commands
		switch strings.ToLower(input) {
		case "exit", "quit":
			fmt.Println("Goodbye!")
			return
			
		case "help":
			fmt.Println("\nCommands:")
			fmt.Println("  save [title]  - Save the last response as a corpus document")
			fmt.Println("  clear        - Clear conversation history")
			fmt.Println("  help         - Show this help")
			fmt.Println("  exit/quit    - Exit the chat")
			fmt.Println()
			continue
			
		case "clear":
			corpusAgent.ClearHistory()
			fmt.Println("Conversation history cleared.")
			continue
			
		case "":
			continue
		}
		
		// Handle save command
		if strings.HasPrefix(strings.ToLower(input), "save") {
			if lastResponse == "" {
				fmt.Println("No response to save. Ask a question first.")
				continue
			}
			
			parts := strings.SplitN(input, " ", 2)
			title := "Generated Document"
			if len(parts) > 1 {
				title = parts[1]
			}
			
			// Create and save document
			doc := &corpus.CorpusDoc{
				Title:     title,
				Body:      lastResponse,
				Tags:      []string{"generated", "corpus-agent", "chat"},
				Source:    "corpus-agent-chat",
				GuildID:   "corpus",
				AgentID:   corpusAgent.GetID(),
			}
			
			if err := corpusAgent.SaveGeneratedDocument(ctx, doc); err != nil {
				fmt.Printf("Error saving document: %v\n", err)
			} else {
				fmt.Printf("Document saved as: %s\n", title)
			}
			continue
		}
		
		// Process query
		fmt.Println("\nSearching knowledge base...")
		response, err := corpusAgent.Execute(ctx, input)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
		
		// Display response
		fmt.Println("\n" + response)
		fmt.Println()
		
		lastResponse = response
	}
	
	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading input: %v\n", err)
	}
}

func initializeCorpusAgent(providerType, model string) (*corpusagent.CorpusAgent, error) {
	// Get corpus configuration
	cfg, err := getCorpusConfig()
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get corpus configuration").
			WithComponent("cli").
			WithOperation("corpus.query.initializeCorpusAgent")
	}
	
	// Create AI provider
	var provider interfaces.AIProvider
	factory := providers.NewFactoryV2()
	
	if providerType != "" {
		// Use specified provider
		var pType providers.ProviderType
		switch strings.ToLower(providerType) {
		case "ollama":
			pType = providers.ProviderOllama
		case "openai":
			pType = providers.ProviderOpenAI
		case "anthropic":
			pType = providers.ProviderAnthropic
		default:
			return nil, gerror.New(gerror.ErrCodeInvalidInput, "unsupported provider type", nil).
				WithComponent("cli").
				WithOperation("corpus.query.initializeCorpusAgent").
				WithDetails("provider_type", providerType)
		}
		
		// Get API key or base URL
		apiKey := ""
		if pType == providers.ProviderOllama {
			apiKey = os.Getenv("OLLAMA_HOST")
			if apiKey == "" {
				apiKey = "http://localhost:11434"
			}
		} else {
			envKey := fmt.Sprintf("%s_API_KEY", strings.ToUpper(providerType))
			apiKey = os.Getenv(envKey)
			if apiKey == "" {
				return nil, gerror.New(gerror.ErrCodeProviderAuth, "missing API key", nil).
					WithComponent("cli").
					WithOperation("corpus.query.initializeCorpusAgent").
					WithDetails("env_key", envKey).
					WithDetails("provider_type", providerType)
			}
		}
		
		provider, err = factory.CreateAIProvider(pType, apiKey)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeProvider, "failed to create provider").
				WithComponent("cli").
				WithOperation("corpus.query.initializeCorpusAgent").
				WithDetails("provider_type", providerType)
		}
	} else {
		// Auto-detect provider from vector factory
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "provider type required for corpus agent", nil).
			WithComponent("cli").
			WithOperation("corpus.query.initializeCorpusAgent")
	}
	
	// Create vector store configuration
	vectorConfig := &vector.StoreConfig{
		Type:              vector.StoreTypeChromem,
		EmbeddingProvider: provider,
		EmbeddingModel:    model,
		ChromemConfig: vector.ChromemConfig{
			PersistencePath:   filepath.Join(cfg.CorpusPath, "..", "embeddings"),
			DefaultCollection: "corpus",
		},
	}
	
	// Create vector store
	vectorStore, err := vector.NewVectorStore(context.Background(), vectorConfig)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create vector store").
			WithComponent("cli").
			WithOperation("corpus.query.initializeCorpusAgent")
	}
	
	// Create RAG configuration
	ragConfig := rag.Config{
		ChunkSize:    1000,
		ChunkOverlap: 200,
		MaxResults:   10,
		UseCorpus:    true,
		CorpusPath:   cfg.CorpusPath,
	}
	
	// Create retriever
	retriever := rag.NewRetrieverWithStore(vectorStore, ragConfig)
	
	// Create Corpus Agent
	corpusAgent := corpusagent.NewCorpusAgent(retriever, provider, cfg)
	
	return corpusAgent, nil
}

func generateTitle(query string) string {
	// Simple title generation from query
	words := strings.Fields(query)
	if len(words) > 5 {
		words = words[:5]
	}
	
	title := strings.Join(words, " ")
	
	// Remove question marks
	title = strings.TrimSuffix(title, "?")
	
	// Capitalize first letter
	if len(title) > 0 {
		title = strings.ToUpper(title[:1]) + title[1:]
	}
	
	return title
}

func init() {
	// Add commands to corpus
	corpusCmd.AddCommand(corpusQueryCmd)
	corpusCmd.AddCommand(corpusChatCmd)
	
	// Add flags to query command
	corpusQueryCmd.Flags().BoolP("save", "s", false, "Save response as corpus document")
	corpusQueryCmd.Flags().StringP("title", "t", "", "Title for saved document")
	corpusQueryCmd.Flags().StringP("provider", "p", "", "AI provider (ollama, openai, anthropic)")
	corpusQueryCmd.Flags().StringP("model", "m", "", "Model to use for generation")
	
	// Add flags to chat command
	corpusChatCmd.Flags().StringP("provider", "p", "", "AI provider (ollama, openai, anthropic)")
	corpusChatCmd.Flags().StringP("model", "m", "", "Model to use for generation")
}