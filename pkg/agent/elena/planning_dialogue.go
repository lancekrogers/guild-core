// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package elena

import (
	"context"
	"fmt"
	"strings"

	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// PlanningStage represents the current stage of commission planning
type PlanningStage int

const (
	StageIntroduction PlanningStage = iota
	StageProjectPurpose
	StageProjectType
	StageTechnology
	StageRequirements
	StageConstraints
	StageTeamSize
	StageTimeline
	StageSummary
	StageComplete
)

// PlanningDialogue manages Elena's commission planning conversation
type PlanningDialogue struct {
	ID          string
	stage       PlanningStage
	context     map[string]interface{}
	questions   []Question
	responses   map[string]string
	currentPath string // tracks conversation path for dynamic branching
}

// Question represents a planning question
type Question struct {
	ID       string
	Stage    PlanningStage
	Text     string
	Type     string // "open", "choice", "confirmation"
	Choices  []string
	Required bool
	FollowUp func(response string) *Question // Dynamic follow-up based on answer
}

// NewPlanningDialogue creates a new planning dialogue session
func NewPlanningDialogue(id string) *PlanningDialogue {
	return &PlanningDialogue{
		ID:        id,
		stage:     StageIntroduction,
		context:   make(map[string]interface{}),
		responses: make(map[string]string),
		questions: []Question{},
	}
}

// GetNextQuestion returns Elena's next question based on current stage and context
func (pd *PlanningDialogue) GetNextQuestion() string {
	switch pd.stage {
	case StageIntroduction:
		return pd.getIntroduction()
	case StageProjectPurpose:
		return pd.askProjectPurpose()
	case StageProjectType:
		return pd.askProjectType()
	case StageTechnology:
		return pd.askTechnology()
	case StageRequirements:
		return pd.askRequirements()
	case StageConstraints:
		return pd.askConstraints()
	case StageTeamSize:
		return pd.askTeamSize()
	case StageTimeline:
		return pd.askTimeline()
	case StageSummary:
		return pd.showSummary()
	case StageComplete:
		return "Our commission planning is complete! I shall now craft a comprehensive document from our discussion."
	default:
		return "I seem to have lost my place in our planning. Let us begin anew with '/commission new'."
	}
}

// ProcessResponse processes user's response and advances the dialogue
func (pd *PlanningDialogue) ProcessResponse(ctx context.Context, response string) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("elena.planning").
			WithOperation("ProcessResponse")
	}

	// Store the response
	pd.responses[pd.getCurrentQuestionID()] = response

	// Process based on current stage
	switch pd.stage {
	case StageIntroduction:
		pd.stage = StageProjectPurpose
	case StageProjectPurpose:
		pd.processPurpose(response)
	case StageProjectType:
		pd.processProjectType(response)
	case StageTechnology:
		pd.stage = StageRequirements
	case StageRequirements:
		pd.stage = StageConstraints
	case StageConstraints:
		pd.stage = StageTeamSize
	case StageTeamSize:
		pd.stage = StageTimeline
	case StageTimeline:
		pd.stage = StageSummary
	case StageSummary:
		if strings.ToLower(response) == "yes" || strings.Contains(strings.ToLower(response), "proceed") {
			pd.stage = StageComplete
		} else {
			// Allow editing
			pd.handleSummaryEdit(response)
		}
	}

	return nil
}

// getIntroduction returns Elena's introduction
func (pd *PlanningDialogue) getIntroduction() string {
	return `Greetings, noble artisan! I am Elena, Guild Master of this distinguished company. 

I shall guide thee through planning thy commission - a quest most worthy! Together we shall craft a document that clearly defines thy project's purpose, requirements, and constraints.

What manner of creation dost thou wish to bring forth? Describe thy vision, and I shall help thee shape it into a proper commission.`
}

// askProjectPurpose asks about the project's primary purpose
func (pd *PlanningDialogue) askProjectPurpose() string {
	return `Excellent! Before we delve into specifics, I must understand thy primary purpose.

Art thou seeking to:
1. **Build software** - Create applications, services, or tools
2. **Conduct deep research** - Analyze systems, investigate problems, or explore technologies
3. **Improve existing systems** - Refactor, optimize, or enhance current solutions
4. **Other purpose** - Something else entirely

What is thy primary intent, good artisan?`
}

// askProjectType asks about specific project type based on purpose
func (pd *PlanningDialogue) askProjectType() string {
	purpose := pd.responses["project_purpose"]
	projectDesc := pd.responses["project_description"]

	// Dynamic questioning based on previous responses
	if strings.Contains(strings.ToLower(purpose), "build software") {
		if strings.Contains(strings.ToLower(projectDesc), "api") {
			return `I perceive thou wishest to forge an API! A wise choice for enabling communication between systems.

What manner of API dost thou envision?
- **REST API** - Traditional HTTP-based interface
- **GraphQL API** - Flexible query language for thy data
- **gRPC Service** - High-performance RPC framework
- **WebSocket API** - Real-time bidirectional communication

Which architectural style suits thy needs?`
		} else if strings.Contains(strings.ToLower(projectDesc), "web") {
			return `A web application! A canvas for both artistry and function.

What components shall thy web application require?
- **Frontend only** - Client-side application
- **Backend only** - Server-side API or service  
- **Full-stack** - Both frontend and backend united
- **Static site** - Content without dynamic server logic

Which architecture befits thy vision?`
		} else if strings.Contains(strings.ToLower(projectDesc), "cli") || strings.Contains(strings.ToLower(projectDesc), "command") {
			return `A command-line tool! The choice of those who value efficiency and automation.

What shall be the primary function of thy CLI tool?
- **Development tool** - Aid in building software
- **System utility** - Manage or monitor systems
- **Data processing** - Transform or analyze information
- **Automation** - Orchestrate tasks and workflows

Which purpose aligns with thy intent?`
		}
	} else if strings.Contains(strings.ToLower(purpose), "research") {
		return `Deep research requires methodical investigation and clear documentation.

What is the focus of thy research?
- **Technology evaluation** - Assess tools, frameworks, or platforms
- **Architecture analysis** - Study system designs and patterns
- **Performance investigation** - Analyze bottlenecks and optimizations
- **Security audit** - Examine vulnerabilities and protections
- **Market analysis** - Research solutions and competitors

Which domain calls for thy scholarly attention?`
	}

	// Default question if no specific match
	return `Tell me more about the specific type of project thou envisions. 

Is it perhaps:
- A service that processes data?
- An application with user interface?
- A library or framework for other developers?
- A system integration project?
- Something else entirely?

Describe thy project type in thine own words.`
}

// askTechnology asks about technology choices
func (pd *PlanningDialogue) askTechnology() string {
	// Check if we already have technology context
	if tech, ok := pd.context["detected_technology"]; ok {
		return fmt.Sprintf(`I notice thy workspace already employs %s. 

Shall we continue with this technology stack, or dost thou wish to specify different tools for this commission?

Please list thy technology preferences:
- **Programming language(s)**
- **Frameworks and libraries**
- **Database systems** (if needed)
- **External services** (if any)

What technologies shall we employ?`, tech)
	}

	return `Now, let us discuss the tools of thy craft - the technologies thou shall wield.

Please specify:
- **Primary programming language** (e.g., Go, Python, TypeScript, Rust)
- **Key frameworks or libraries** (e.g., React, Django, Gin, Express)
- **Data storage needs** (e.g., PostgreSQL, Redis, MongoDB, none)
- **External integrations** (e.g., APIs, cloud services, third-party tools)

Share thy technology choices, and I shall ensure our commission reflects them properly.`
}

// askRequirements asks about specific requirements
func (pd *PlanningDialogue) askRequirements() string {
	return fmt.Sprintf(`Now we must define the specific requirements for %s.

Think carefully about:
- **Core features** - What must it do?
- **User interactions** - How will it be used?
- **Data handling** - What information will it process?
- **Integration points** - What must it connect with?
- **Success criteria** - How will we know 'tis complete?

List thy requirements, and be as specific as thy vision allows. I shall help organize them into a proper structure.`, 
		pd.getProjectReference())
}

// askConstraints asks about constraints and limitations
func (pd *PlanningDialogue) askConstraints() string {
	return `Every worthy quest has its challenges and boundaries. Let us identify thine.

Consider these constraints:
- **Performance requirements** (response time, throughput, scalability)
- **Security considerations** (authentication, data protection, compliance)
- **Resource limitations** (budget, infrastructure, team expertise)
- **Compatibility needs** (browsers, platforms, versions)
- **Non-functional requirements** (maintainability, documentation, testing)

What constraints must we respect in this endeavor?`
}

// askTeamSize asks about team size and collaboration
func (pd *PlanningDialogue) askTeamSize() string {
	return `Tell me of thy company - who shall undertake this commission?

- **Solo artisan** - Working alone with focus and freedom
- **Small team** (2-3 people) - Close collaboration with shared vision
- **Medium team** (4-8 people) - Structured coordination needed
- **Large team** (9+ people) - Formal processes and clear boundaries

Understanding thy team helps me structure the commission appropriately. How many shall labor on this quest?`
}

// askTimeline asks about project timeline
func (pd *PlanningDialogue) askTimeline() string {
	return `Time is a resource most precious. What timeline governs this commission?

- **Exploratory** (1-2 weeks) - Proof of concept or investigation
- **Sprint** (2-4 weeks) - Focused delivery of core features
- **Project** (1-3 months) - Full implementation with iterations
- **Program** (3+ months) - Major initiative with multiple phases

Additionally, are there:
- Specific deadlines or milestones?
- Dependencies on other work?
- Phases thou wishest to define?

Share thy temporal constraints, and I shall factor them into our planning.`
}

// showSummary shows a summary of collected information
func (pd *PlanningDialogue) showSummary() string {
	var sb strings.Builder
	sb.WriteString("📜 **Commission Summary**\n\n")
	sb.WriteString("Based on our discourse, here is what I have gleaned:\n\n")
	
	// Project overview
	sb.WriteString("**🎯 Project Vision:**\n")
	sb.WriteString(fmt.Sprintf("- Description: %s\n", pd.responses["project_description"]))
	sb.WriteString(fmt.Sprintf("- Purpose: %s\n", pd.responses["project_purpose"]))
	sb.WriteString(fmt.Sprintf("- Type: %s\n", pd.responses["project_type"]))
	sb.WriteString("\n")
	
	// Technology
	if tech, ok := pd.responses["technology"]; ok && tech != "" {
		sb.WriteString("**🛠 Technology Stack:**\n")
		sb.WriteString(fmt.Sprintf("%s\n\n", pd.formatList(tech)))
	}
	
	// Requirements
	if reqs, ok := pd.responses["requirements"]; ok && reqs != "" {
		sb.WriteString("**📋 Key Requirements:**\n")
		sb.WriteString(fmt.Sprintf("%s\n\n", pd.formatList(reqs)))
	}
	
	// Constraints
	if constraints, ok := pd.responses["constraints"]; ok && constraints != "" {
		sb.WriteString("**⚠️ Constraints:**\n")
		sb.WriteString(fmt.Sprintf("%s\n\n", pd.formatList(constraints)))
	}
	
	// Team and Timeline
	sb.WriteString("**👥 Team & Timeline:**\n")
	sb.WriteString(fmt.Sprintf("- Team Size: %s\n", pd.responses["team_size"]))
	sb.WriteString(fmt.Sprintf("- Timeline: %s\n", pd.responses["timeline"]))
	sb.WriteString("\n")
	
	sb.WriteString("---\n\n")
	sb.WriteString("Dost this summary capture thy vision accurately? \n\n")
	sb.WriteString("- Type **'yes'** to proceed with commission generation\n")
	sb.WriteString("- Or tell me what needs adjustment, and I shall revise accordingly")
	
	return sb.String()
}

// Helper methods

func (pd *PlanningDialogue) getCurrentQuestionID() string {
	stageNames := []string{
		"introduction",
		"project_purpose", 
		"project_type",
		"technology",
		"requirements",
		"constraints",
		"team_size",
		"timeline",
		"summary",
		"complete",
	}
	
	if int(pd.stage) < len(stageNames) {
		return stageNames[pd.stage]
	}
	return "unknown"
}

func (pd *PlanningDialogue) getProjectReference() string {
	if desc, ok := pd.responses["project_description"]; ok && desc != "" {
		// Extract a short reference from the description
		words := strings.Fields(desc)
		if len(words) > 5 {
			return strings.Join(words[:5], " ") + "..."
		}
		return desc
	}
	return "thy project"
}

func (pd *PlanningDialogue) formatList(text string) string {
	lines := strings.Split(text, "\n")
	var formatted []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			if !strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "*") {
				line = "- " + line
			}
			formatted = append(formatted, line)
		}
	}
	return strings.Join(formatted, "\n")
}

func (pd *PlanningDialogue) processPurpose(response string) {
	lower := strings.ToLower(response)
	if strings.Contains(lower, "build") || strings.Contains(lower, "software") || strings.Contains(lower, "1") {
		pd.context["purpose_category"] = "build_software"
		pd.stage = StageProjectType
	} else if strings.Contains(lower, "research") || strings.Contains(lower, "2") {
		pd.context["purpose_category"] = "deep_research"
		pd.stage = StageProjectType
	} else if strings.Contains(lower, "improve") || strings.Contains(lower, "existing") || strings.Contains(lower, "3") {
		pd.context["purpose_category"] = "improve_existing"
		pd.stage = StageProjectType
	} else {
		pd.context["purpose_category"] = "other"
		pd.stage = StageProjectType
	}
}

func (pd *PlanningDialogue) processProjectType(response string) {
	// Store the project type and determine if we need technology info
	pd.context["project_type_detail"] = response
	pd.stage = StageTechnology
}

func (pd *PlanningDialogue) handleSummaryEdit(response string) {
	lower := strings.ToLower(response)
	
	// Check what they want to edit
	if strings.Contains(lower, "tech") || strings.Contains(lower, "language") {
		pd.stage = StageTechnology
	} else if strings.Contains(lower, "require") {
		pd.stage = StageRequirements
	} else if strings.Contains(lower, "constraint") {
		pd.stage = StageConstraints
	} else if strings.Contains(lower, "team") {
		pd.stage = StageTeamSize
	} else if strings.Contains(lower, "time") {
		pd.stage = StageTimeline
	} else {
		// Stay in summary and ask for clarification
		pd.context["needs_clarification"] = true
	}
}

// GetResponses returns all collected responses
func (pd *PlanningDialogue) GetResponses() map[string]string {
	return pd.responses
}

// GetContext returns the dialogue context
func (pd *PlanningDialogue) GetContext() map[string]interface{} {
	return pd.context
}

// IsComplete returns true if the dialogue is complete
func (pd *PlanningDialogue) IsComplete() bool {
	return pd.stage == StageComplete
}

// SetProjectContext sets initial project context (e.g., detected language)
func (pd *PlanningDialogue) SetProjectContext(key string, value interface{}) {
	pd.context[key] = value
}