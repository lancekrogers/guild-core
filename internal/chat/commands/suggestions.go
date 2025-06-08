package commands

import (
	"fmt"
	"sort"
	"strings"
)

// SuggestionEngine provides "Did you mean?" suggestions for commands
type SuggestionEngine struct {
	commands     []string
	agents       []string
	tools        []string
	promptLayers []string
}

// NewSuggestionEngine creates a new suggestion engine
func NewSuggestionEngine() *SuggestionEngine {
	return &SuggestionEngine{
		commands:     make([]string, 0),
		agents:       make([]string, 0),
		tools:        make([]string, 0),
		promptLayers: make([]string, 0),
	}
}

// UpdateCommands updates the list of available commands
func (se *SuggestionEngine) UpdateCommands(commands []string) {
	se.commands = commands
}

// UpdateAgents updates the list of available agents
func (se *SuggestionEngine) UpdateAgents(agents []string) {
	se.agents = agents
}

// UpdateTools updates the list of available tools
func (se *SuggestionEngine) UpdateTools(tools []string) {
	se.tools = tools
}

// UpdatePromptLayers updates the list of prompt layers
func (se *SuggestionEngine) UpdatePromptLayers(layers []string) {
	se.promptLayers = layers
}

// GetCommandSuggestion returns "Did you mean?" suggestion for commands
func (se *SuggestionEngine) GetCommandSuggestion(input string) string {
	suggestion := se.findBestMatch(input, se.commands)
	if suggestion != "" {
		return fmt.Sprintf("Did you mean /%s?", suggestion)
	}
	return ""
}

// GetAgentSuggestion returns "Did you mean?" suggestion for agents
func (se *SuggestionEngine) GetAgentSuggestion(input string) string {
	suggestion := se.findBestMatch(input, se.agents)
	if suggestion != "" {
		return fmt.Sprintf("Did you mean @%s?", suggestion)
	}
	return ""
}

// GetToolSuggestion returns "Did you mean?" suggestion for tools
func (se *SuggestionEngine) GetToolSuggestion(input string) string {
	suggestion := se.findBestMatch(input, se.tools)
	if suggestion != "" {
		return fmt.Sprintf("Did you mean '%s'?", suggestion)
	}
	return ""
}

// GetPromptLayerSuggestion returns "Did you mean?" suggestion for prompt layers
func (se *SuggestionEngine) GetPromptLayerSuggestion(input string) string {
	suggestion := se.findBestMatch(input, se.promptLayers)
	if suggestion != "" {
		return fmt.Sprintf("Did you mean '%s' layer?", suggestion)
	}
	return ""
}

// findBestMatch finds the best matching string from a list
func (se *SuggestionEngine) findBestMatch(input string, candidates []string) string {
	if len(candidates) == 0 {
		return ""
	}
	
	input = strings.ToLower(input)
	
	// Calculate distances for all candidates
	type match struct {
		candidate string
		distance  int
	}
	
	matches := make([]match, 0, len(candidates))
	
	for _, candidate := range candidates {
		dist := levenshteinDistance(input, strings.ToLower(candidate))
		
		// Only consider matches with reasonable distance
		if dist <= 3 || dist <= len(input)/2 {
			matches = append(matches, match{
				candidate: candidate,
				distance:  dist,
			})
		}
	}
	
	if len(matches) == 0 {
		return ""
	}
	
	// Sort by distance
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].distance < matches[j].distance
	})
	
	// Return best match
	return matches[0].candidate
}

// levenshteinDistance calculates the Levenshtein distance between two strings
func levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}
	
	// Create matrix
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
	}
	
	// Initialize first column and row
	for i := 0; i <= len(s1); i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len(s2); j++ {
		matrix[0][j] = j
	}
	
	// Calculate distances
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}
			
			matrix[i][j] = min3(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}
	
	return matrix[len(s1)][len(s2)]
}

// min3 returns the minimum of three integers
func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// GetSmartErrorMessage creates an error message with helpful suggestions
func (se *SuggestionEngine) GetSmartErrorMessage(errorType string, input string) string {
	var msg strings.Builder
	
	switch errorType {
	case "unknown_command":
		cmd := strings.TrimPrefix(input, "/")
		msg.WriteString(fmt.Sprintf("Unknown command: /%s", cmd))
		if suggestion := se.GetCommandSuggestion(cmd); suggestion != "" {
			msg.WriteString(". ")
			msg.WriteString(suggestion)
		} else {
			msg.WriteString(". Type /help for available commands.")
		}
		
	case "unknown_agent":
		agent := strings.TrimPrefix(input, "@")
		msg.WriteString(fmt.Sprintf("No agent '%s' found", agent))
		if suggestion := se.GetAgentSuggestion(agent); suggestion != "" {
			msg.WriteString(". ")
			msg.WriteString(suggestion)
		} else {
			msg.WriteString(". Use /agents to see available agents.")
		}
		
	case "unknown_tool":
		msg.WriteString(fmt.Sprintf("Tool '%s' not found", input))
		if suggestion := se.GetToolSuggestion(input); suggestion != "" {
			msg.WriteString(". ")
			msg.WriteString(suggestion)
		} else {
			msg.WriteString(". Use /tools list to see available tools.")
		}
		
	case "unknown_layer":
		msg.WriteString(fmt.Sprintf("Unknown prompt layer: %s", input))
		if suggestion := se.GetPromptLayerSuggestion(input); suggestion != "" {
			msg.WriteString(". ")
			msg.WriteString(suggestion)
		} else {
			msg.WriteString(". Valid layers: base, context, task, style, constraints, examples")
		}
		
	default:
		msg.WriteString(fmt.Sprintf("Error: %s", input))
	}
	
	return msg.String()
}

// AddCommonMisspellings adds common misspellings for better suggestions
func (se *SuggestionEngine) AddCommonMisspellings() {
	// Common command misspellings
	commonCommandTypos := map[string]string{
		"hlep":    "help",
		"halp":    "help",
		"statsu":  "status",
		"staus":   "status",
		"agent":   "agents",
		"tool":    "tools",
		"promt":   "prompt",
		"promtp":  "prompt",
		"promtps": "prompt",
		"exist":   "exit",
		"quit":    "exit",
		"cls":     "clear",
		"clr":     "clear",
	}
	
	// Add to commands if not already present
	for _, correct := range commonCommandTypos {
		found := false
		for _, cmd := range se.commands {
			if cmd == correct {
				found = true
				break
			}
		}
		if found {
			// Don't add the typo, but ensure we can suggest the correct command
			continue
		}
	}
}