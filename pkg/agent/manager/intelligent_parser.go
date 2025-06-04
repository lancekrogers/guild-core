package manager

import (
	"context"
	"fmt"

	"github.com/guild-ventures/guild-core/pkg/prompts"
)

// IntelligentParser uses either pattern matching or LLM extraction based on configuration
type IntelligentParser struct {
	patternParser *ResponseParserImpl
	taskExtractor *TaskExtractor
	useExtractor  bool
	mode          ParserMode
}

// ParserMode defines how tasks should be extracted
type ParserMode string

const (
	// ParserModePattern uses regex patterns (fast, deterministic)
	ParserModePattern ParserMode = "pattern"
	
	// ParserModeExtractor uses LLM intelligence (flexible, adaptive)
	ParserModeExtractor ParserMode = "extractor"
	
	// ParserModeAuto tries extractor first, falls back to patterns
	ParserModeAuto ParserMode = "auto"
)

// IntelligentParserConfig configures the parser behavior
type IntelligentParserConfig struct {
	Mode          ParserMode
	ArtisanClient ArtisanClient
	PromptManager prompts.LayeredManager
}

// NewIntelligentParser creates a parser that can use multiple extraction strategies
func NewIntelligentParser(config IntelligentParserConfig) *IntelligentParser {
	parser := &IntelligentParser{
		patternParser: NewResponseParser(),
		mode:          config.Mode, // Store the configured mode
	}
	
	// Set up based on mode
	switch config.Mode {
	case ParserModeExtractor:
		if config.ArtisanClient != nil && config.PromptManager != nil {
			parser.taskExtractor = NewTaskExtractor(config.ArtisanClient, config.PromptManager)
			parser.useExtractor = true
		} else {
			// Fall back to pattern mode if extractor requirements not met
			parser.useExtractor = false
			parser.mode = ParserModePattern // Update mode to reflect fallback
		}
		
	case ParserModeAuto:
		// Set up both parsers
		if config.ArtisanClient != nil && config.PromptManager != nil {
			parser.taskExtractor = NewTaskExtractor(config.ArtisanClient, config.PromptManager)
		}
		parser.useExtractor = true
		
	default: // ParserModePattern
		parser.useExtractor = false
	}
	
	return parser
}

// ParseResponse implements the ResponseParser interface with intelligent routing
func (ip *IntelligentParser) ParseResponse(ctx context.Context, response *ArtisanResponse) (*FileStructure, error) {
	// First try pattern-based parsing for structure
	structure, err := ip.patternParser.ParseResponse(response)
	if err != nil && !ip.useExtractor {
		return nil, err
	}
	
	// If we have a structure but should use extractor for tasks
	if ip.useExtractor && ip.taskExtractor != nil && structure != nil {
		// Create a refined commission from the parsed structure
		refined := &RefinedCommission{
			CommissionID: "temp-extraction",
			Structure:    structure,
			Metadata:     response.Metadata,
		}
		
		// Extract tasks using LLM intelligence
		extractionResult, extractErr := ip.taskExtractor.ExtractTasks(ctx, refined)
		if extractErr == nil && extractionResult != nil {
			// Convert extracted tasks to TaskInfo format
			tasks := ConvertExtractionResultToTaskInfos(extractionResult)
			
			// Update the file metadata with intelligently extracted tasks
			for _, file := range structure.Files {
				file.Metadata["tasks"] = tasks
				file.TasksCount = len(tasks)
			}
			
			return structure, nil
		}
		
		// If extraction failed but we're in auto mode, keep pattern results
		if ip.useExtractor && structure != nil {
			return structure, nil
		}
		
		// Otherwise return the extraction error
		if extractErr != nil {
			return nil, fmt.Errorf("task extraction failed: %w", extractErr)
		}
	}
	
	return structure, err
}

// ExtractTasksDirectly uses only the LLM extractor for maximum flexibility
func (ip *IntelligentParser) ExtractTasksDirectly(ctx context.Context, refinedCommission *RefinedCommission) ([]TaskInfo, error) {
	if ip.taskExtractor == nil {
		return nil, fmt.Errorf("task extractor not configured")
	}
	
	result, err := ip.taskExtractor.ExtractTasks(ctx, refinedCommission)
	if err != nil {
		return nil, err
	}
	
	return ConvertExtractionResultToTaskInfos(result), nil
}

// GetExtractionMode returns the current extraction mode
func (ip *IntelligentParser) GetExtractionMode() ParserMode {
	// For auto mode, return the actual mode being used
	if ip.mode == ParserModeAuto {
		if ip.taskExtractor != nil && ip.useExtractor {
			return ParserModeExtractor
		}
		return ParserModePattern
	}
	// For other modes, return the configured/fallback mode
	return ip.mode
}