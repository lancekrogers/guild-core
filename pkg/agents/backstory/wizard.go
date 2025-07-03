// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package backstory

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/lancekrogers/guild/pkg/agents/backstory/templates"
	"github.com/lancekrogers/guild/pkg/config"
	"github.com/lancekrogers/guild/pkg/gerror"
)

// AgentCreationWizard provides an interactive way to create agents with rich backstories
type AgentCreationWizard struct {
	templates map[string]*config.AgentConfig
}

// NewAgentCreationWizard creates a new agent creation wizard
func NewAgentCreationWizard() *AgentCreationWizard {
	return &AgentCreationWizard{
		templates: templates.SpecialistTemplates,
	}
}

// WizardResult contains the result of the wizard
type WizardResult struct {
	Agent      *config.AgentConfig
	IsTemplate bool
	Customized bool
}

// RunInteractiveWizard runs the full interactive agent creation process
func (w *AgentCreationWizard) RunInteractiveWizard() (*WizardResult, error) {
	fmt.Println("🎭 Welcome to the Guild Agent Creation Wizard!")
	fmt.Println("==============================================")
	fmt.Println("Create your own skilled artisan to join the digital guild.")
	fmt.Println()

	// Step 1: Choose creation method
	method, err := w.chooseCreationMethod()
	if err != nil {
		return nil, err
	}

	var agent *config.AgentConfig
	var isTemplate bool
	var customized bool

	switch method {
	case "template":
		agent, err = w.selectAndCustomizeTemplate()
		isTemplate = true
		customized = true
		if err != nil {
			return nil, err
		}
	case "scratch":
		agent, err = w.createFromScratch()
		isTemplate = false
		if err != nil {
			return nil, err
		}
	case "browse":
		w.browseTemplates()
		return w.RunInteractiveWizard() // Start over after browsing
	default:
		return nil, gerror.New(gerror.ErrCodeValidation, "invalid creation method", nil)
	}

	// Step 2: Review and confirm
	if w.shouldReviewAgent() {
		w.displayAgentSummary(agent)
		if w.shouldMakeChanges() {
			// Allow basic modifications
			agent, err = w.makeBasicModifications(agent)
			if err != nil {
				return nil, err
			}
			customized = true
		}
	}

	return &WizardResult{
		Agent:      agent,
		IsTemplate: isTemplate,
		Customized: customized,
	}, nil
}

// chooseCreationMethod lets the user choose how to create their agent
func (w *AgentCreationWizard) chooseCreationMethod() (string, error) {
	fmt.Println("How would you like to create your agent?")
	fmt.Println("1. 📋 Use a specialist template (recommended)")
	fmt.Println("2. ⚡ Create from scratch")
	fmt.Println("3. 👀 Browse available templates first")
	fmt.Println()

	for {
		fmt.Print("Enter your choice (1-3): ")
		var choice string
		fmt.Scanln(&choice)

		switch choice {
		case "1":
			return "template", nil
		case "2":
			return "scratch", nil
		case "3":
			return "browse", nil
		default:
			fmt.Println("❌ Please enter 1, 2, or 3")
		}
	}
}

// browseTemplates displays all available templates
func (w *AgentCreationWizard) browseTemplates() {
	fmt.Println("\n📚 Available Specialist Templates:")
	fmt.Println("===================================")

	for key, template := range w.templates {
		fmt.Printf("\n🏷️  %s\n", key)
		fmt.Printf("   Name: %s\n", template.Name)
		fmt.Printf("   Role: %s\n", template.Type)
		fmt.Printf("   Description: %s\n", template.Description)

		if template.Backstory != nil {
			fmt.Printf("   Rank: %s\n", template.Backstory.GuildRank)
			if len(template.Backstory.Specialties) > 0 {
				fmt.Printf("   Specialties: %s\n", strings.Join(template.Backstory.Specialties, ", "))
			}
		}
	}

	fmt.Println("\n📖 Guild Master Template:")
	guildMaster := templates.CreateMedievalGuildMaster()
	fmt.Printf("   Name: %s\n", guildMaster.Name)
	fmt.Printf("   Role: %s\n", guildMaster.Type)
	fmt.Printf("   Description: %s\n", guildMaster.Description)

	fmt.Println("\n" + strings.Repeat("-", 50))
	fmt.Print("Press Enter to continue...")
	fmt.Scanln()
}

// selectAndCustomizeTemplate helps user select and customize a template
func (w *AgentCreationWizard) selectAndCustomizeTemplate() (*config.AgentConfig, error) {
	fmt.Println("\n🎯 Select a Specialist Template:")
	fmt.Println("=================================")

	// List available templates
	templates := []struct {
		key         string
		displayName string
		agent       *config.AgentConfig
	}{
		{"security-sentinel", "Security Guardian", w.templates["security-sentinel"]},
		{"performance-artisan", "Performance Optimizer", w.templates["performance-artisan"]},
		{"frontend-artist", "UX Designer", w.templates["frontend-artist"]},
		{"code-sage", "Code Architect", w.templates["code-sage"]},
		{"data-mystic", "Data Scientist", w.templates["data-mystic"]},
		{"guild-master", "Guild Master", templates.CreateMedievalGuildMaster()},
	}

	for i, tmpl := range templates {
		fmt.Printf("%d. %s (%s)\n", i+1, tmpl.displayName, tmpl.agent.Name)
		fmt.Printf("   %s\n", tmpl.agent.Description)
		fmt.Println()
	}

	for {
		fmt.Print("Select template (1-6): ")
		var choice string
		fmt.Scanln(&choice)

		choiceNum, err := strconv.Atoi(choice)
		if err != nil || choiceNum < 1 || choiceNum > len(templates) {
			fmt.Println("❌ Please enter a number between 1 and 6")
			continue
		}

		selectedTemplate := templates[choiceNum-1]
		fmt.Printf("\n✅ Selected: %s\n", selectedTemplate.displayName)

		// Make a copy of the template for customization
		agent := w.copyAgentConfig(selectedTemplate.agent)

		// Ask if they want to customize
		if w.shouldCustomizeTemplate() {
			return w.customizeTemplate(agent)
		}

		return agent, nil
	}
}

// createFromScratch walks through creating an agent from scratch
func (w *AgentCreationWizard) createFromScratch() (*config.AgentConfig, error) {
	fmt.Println("\n⚡ Create Agent from Scratch:")
	fmt.Println("=============================")

	agent := &config.AgentConfig{
		Provider:      "mock", // Default for development
		Model:         "claude-3-sonnet-20240229",
		CostMagnitude: 3,
		Capabilities:  []string{"general_assistance"},
	}

	// Basic information
	agent.ID = w.promptForString("Agent ID (lowercase-with-hyphens):", "my-custom-agent")
	agent.Name = w.promptForString("Agent's full name:", "My Custom Agent")
	agent.Type = w.promptForAgentType()
	agent.Description = w.promptForString("Brief description:", "A skilled artisan")

	// Create backstory
	if w.shouldAddBackstory() {
		agent.Backstory = w.createBackstory()
	}

	// Create personality
	if w.shouldAddPersonality() {
		agent.Personality = w.createPersonality()
	}

	// Create specialization
	if w.shouldAddSpecialization() {
		agent.Specialization = w.createSpecialization()
	}

	return agent, nil
}

// Helper methods for the wizard

func (w *AgentCreationWizard) shouldCustomizeTemplate() bool {
	return w.promptForYesNo("Would you like to customize this template?")
}

func (w *AgentCreationWizard) shouldReviewAgent() bool {
	return w.promptForYesNo("Would you like to review your agent before finishing?")
}

func (w *AgentCreationWizard) shouldMakeChanges() bool {
	return w.promptForYesNo("Would you like to make any changes?")
}

func (w *AgentCreationWizard) shouldAddBackstory() bool {
	return w.promptForYesNo("Add professional backstory and experience?")
}

func (w *AgentCreationWizard) shouldAddPersonality() bool {
	return w.promptForYesNo("Add personality traits and communication style?")
}

func (w *AgentCreationWizard) shouldAddSpecialization() bool {
	return w.promptForYesNo("Add domain specialization and expertise?")
}

func (w *AgentCreationWizard) promptForYesNo(question string) bool {
	for {
		fmt.Printf("%s (y/n): ", question)
		var response string
		fmt.Scanln(&response)

		response = strings.ToLower(strings.TrimSpace(response))
		if response == "y" || response == "yes" {
			return true
		}
		if response == "n" || response == "no" {
			return false
		}
		fmt.Println("❌ Please enter 'y' or 'n'")
	}
}

func (w *AgentCreationWizard) promptForString(prompt, defaultValue string) string {
	fmt.Printf("%s [%s]: ", prompt, defaultValue)
	var response string
	fmt.Scanln(&response)

	if strings.TrimSpace(response) == "" {
		return defaultValue
	}
	return response
}

func (w *AgentCreationWizard) promptForAgentType() string {
	fmt.Println("Select agent type:")
	fmt.Println("1. manager (leads and coordinates)")
	fmt.Println("2. worker (executes tasks)")
	fmt.Println("3. specialist (domain expert)")

	for {
		fmt.Print("Enter choice (1-3): ")
		var choice string
		fmt.Scanln(&choice)

		switch choice {
		case "1":
			return "manager"
		case "2":
			return "worker"
		case "3":
			return "specialist"
		default:
			fmt.Println("❌ Please enter 1, 2, or 3")
		}
	}
}

func (w *AgentCreationWizard) createBackstory() *config.Backstory {
	fmt.Println("\n📚 Creating Agent Backstory:")
	fmt.Println("=============================")

	backstory := &config.Backstory{}

	backstory.Experience = w.promptForString("Years of experience:", "5 years in software development")
	backstory.Expertise = w.promptForString("Core expertise:", "Building reliable systems")
	backstory.Philosophy = w.promptForString("Professional philosophy:", "Quality and user focus")
	backstory.CommunicationStyle = w.promptForString("Communication style:", "Clear and helpful")
	backstory.GuildRank = w.promptForString("Guild rank:", "Journeyman")

	return backstory
}

func (w *AgentCreationWizard) createPersonality() *config.Personality {
	fmt.Println("\n🎭 Creating Agent Personality:")
	fmt.Println("==============================")

	personality := &config.Personality{}

	personality.Formality = w.promptForFormality()
	personality.DetailLevel = w.promptForDetailLevel()
	personality.ApproachStyle = w.promptForApproachStyle()

	// Simple trait scoring
	personality.Assertiveness = w.promptForScale("Assertiveness (1-10):", 5)
	personality.Empathy = w.promptForScale("Empathy (1-10):", 7)
	personality.Patience = w.promptForScale("Patience (1-10):", 6)

	return personality
}

func (w *AgentCreationWizard) createSpecialization() *config.Specialization {
	fmt.Println("\n🔧 Creating Specialization:")
	fmt.Println("===========================")

	spec := &config.Specialization{}

	spec.Domain = w.promptForString("Domain (e.g., 'web-development'):", "general")
	spec.ExpertiseLevel = w.promptForExpertiseLevel()
	spec.Craft = w.promptForString("Medieval craft equivalent:", "Digital Crafting")

	return spec
}

func (w *AgentCreationWizard) promptForFormality() string {
	options := []string{"formal", "casual", "adaptive"}
	return w.promptForOption("Formality level:", options)
}

func (w *AgentCreationWizard) promptForDetailLevel() string {
	options := []string{"concise", "detailed", "adaptive"}
	return w.promptForOption("Detail level:", options)
}

func (w *AgentCreationWizard) promptForApproachStyle() string {
	options := []string{"methodical", "creative", "balanced"}
	return w.promptForOption("Approach style:", options)
}

func (w *AgentCreationWizard) promptForExpertiseLevel() string {
	options := []string{"novice", "intermediate", "expert", "master"}
	return w.promptForOption("Expertise level:", options)
}

func (w *AgentCreationWizard) promptForOption(prompt string, options []string) string {
	fmt.Printf("%s\n", prompt)
	for i, option := range options {
		fmt.Printf("%d. %s\n", i+1, option)
	}

	for {
		fmt.Printf("Enter choice (1-%d): ", len(options))
		var choice string
		fmt.Scanln(&choice)

		choiceNum, err := strconv.Atoi(choice)
		if err != nil || choiceNum < 1 || choiceNum > len(options) {
			fmt.Printf("❌ Please enter a number between 1 and %d\n", len(options))
			continue
		}

		return options[choiceNum-1]
	}
}

func (w *AgentCreationWizard) promptForScale(prompt string, defaultValue int) int {
	for {
		fmt.Printf("%s [%d]: ", prompt, defaultValue)
		var input string
		fmt.Scanln(&input)

		if strings.TrimSpace(input) == "" {
			return defaultValue
		}

		value, err := strconv.Atoi(input)
		if err != nil || value < 1 || value > 10 {
			fmt.Println("❌ Please enter a number between 1 and 10")
			continue
		}

		return value
	}
}

func (w *AgentCreationWizard) customizeTemplate(agent *config.AgentConfig) (*config.AgentConfig, error) {
	fmt.Printf("\n🎨 Customizing %s:\n", agent.Name)
	fmt.Println("==========================================")

	// Allow basic customizations
	newName := w.promptForString("New name (or press Enter to keep current):", agent.Name)
	if newName != agent.Name {
		agent.Name = newName
	}

	newID := w.promptForString("New ID (or press Enter to keep current):", agent.ID)
	if newID != agent.ID {
		agent.ID = newID
	}

	// Allow personality adjustments
	if agent.Personality != nil && w.promptForYesNo("Adjust personality traits?") {
		agent.Personality.Assertiveness = w.promptForScale("Assertiveness (1-10):", agent.Personality.Assertiveness)
		agent.Personality.Empathy = w.promptForScale("Empathy (1-10):", agent.Personality.Empathy)
		agent.Personality.Patience = w.promptForScale("Patience (1-10):", agent.Personality.Patience)
	}

	return agent, nil
}

func (w *AgentCreationWizard) makeBasicModifications(agent *config.AgentConfig) (*config.AgentConfig, error) {
	fmt.Println("\n✏️ What would you like to modify?")
	fmt.Println("1. Name and ID")
	fmt.Println("2. Description")
	fmt.Println("3. Personality traits")
	fmt.Println("4. Nothing, looks good")

	for {
		fmt.Print("Enter choice (1-4): ")
		var choice string
		fmt.Scanln(&choice)

		switch choice {
		case "1":
			agent.Name = w.promptForString("New name:", agent.Name)
			agent.ID = w.promptForString("New ID:", agent.ID)
			return agent, nil
		case "2":
			agent.Description = w.promptForString("New description:", agent.Description)
			return agent, nil
		case "3":
			if agent.Personality != nil {
				agent.Personality.Assertiveness = w.promptForScale("Assertiveness (1-10):", agent.Personality.Assertiveness)
				agent.Personality.Empathy = w.promptForScale("Empathy (1-10):", agent.Personality.Empathy)
				agent.Personality.Patience = w.promptForScale("Patience (1-10):", agent.Personality.Patience)
			}
			return agent, nil
		case "4":
			return agent, nil
		default:
			fmt.Println("❌ Please enter 1, 2, 3, or 4")
		}
	}
}

func (w *AgentCreationWizard) displayAgentSummary(agent *config.AgentConfig) {
	fmt.Printf("\n📋 Agent Summary:\n")
	fmt.Println("=================")
	fmt.Printf("Name: %s\n", agent.Name)
	fmt.Printf("ID: %s\n", agent.ID)
	fmt.Printf("Type: %s\n", agent.Type)
	fmt.Printf("Description: %s\n", agent.Description)

	if agent.Backstory != nil {
		fmt.Printf("Guild Rank: %s\n", agent.Backstory.GuildRank)
		fmt.Printf("Experience: %s\n", agent.Backstory.Experience)
	}

	if agent.Personality != nil {
		fmt.Printf("Personality: Assertiveness %d, Empathy %d, Patience %d\n",
			agent.Personality.Assertiveness,
			agent.Personality.Empathy,
			agent.Personality.Patience)
	}

	if agent.Specialization != nil {
		fmt.Printf("Specialization: %s (%s level)\n",
			agent.Specialization.Domain,
			agent.Specialization.ExpertiseLevel)
	}

	fmt.Println()
}

func (w *AgentCreationWizard) copyAgentConfig(original *config.AgentConfig) *config.AgentConfig {
	// Create a deep copy of the agent config
	copy := *original

	// Copy pointers to avoid shared references
	if original.Backstory != nil {
		backstoryCopy := *original.Backstory
		copy.Backstory = &backstoryCopy
	}

	if original.Personality != nil {
		personalityCopy := *original.Personality
		copy.Personality = &personalityCopy
	}

	if original.Specialization != nil {
		specializationCopy := *original.Specialization
		copy.Specialization = &specializationCopy
	}

	return &copy
}
