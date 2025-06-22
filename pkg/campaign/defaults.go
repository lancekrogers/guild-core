// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package campaign

import (
	"time"
)

// GetDefaultCampaign returns a default campaign for new users
func GetDefaultCampaign() *Campaign {
	now := time.Now()
	return &Campaign{
		ID:          "default-guild-welcome",
		Name:        "Guild Introduction",
		Description: GetDefaultCampaignContent(),
		Status:      CampaignStatusActive,
		Commissions: []string{},
		Tags:        []string{"welcome", "tutorial", "elena", "guild"},
		Metadata: map[string]interface{}{
			"type":       "welcome",
			"manager":    "elena",
			"version":    "1.0",
			"created_by": "guild_init",
		},
		CreatedAt:   now,
		UpdatedAt:   now,
		StartedAt:   &now,
		Progress:    0.0,
	}
}

// GetDefaultCampaignContent provides the Elena-focused welcome experience for new users
func GetDefaultCampaignContent() string {
	return `# Welcome to Your Guild!

*The heavy oak doors of the Guild Hall swing open, revealing a warmly lit chamber adorned with ancient tapestries and maps. At the center stands **Elena**, the Guild Master, her presence both commanding and welcoming.*

## Meet Elena, Your Guild Master

"Greetings, traveler! I am **Elena**, Master of this distinguished guild of artisans. I've been expecting you."

*She gestures to a comfortable chair by the fireplace*

"Please, sit. Let me explain how our guild operates and how we can assist with your ventures..."

---

## The Guild Structure

Elena continues: "Our guild operates much like the great artisan guilds of old. I serve as your **Guild Master** - coordinating our specialists, ensuring quality, and guiding projects to successful completion."

### Your Guild Team:

**Elena (Guild Master)**
- Project coordination and planning
- Task delegation to specialists
- Quality assurance and oversight
- Strategic guidance and recommendations

**Marcus (Backend Specialist)**
- Server architecture and APIs
- Database design and optimization
- Security implementation
- Performance engineering

**Vera (Frontend Specialist)**
- User interface design
- Interactive experiences
- Responsive layouts
- Accessibility standards

*"Each member brings unique expertise. My role is to understand your needs and orchestrate their talents effectively."*

---

## How We Work Together

Elena explains the collaboration process:

"When you bring a commission to our guild, here's what happens:

1. **Initial Consultation** - We discuss your project goals
2. **Planning Session** - I break down the work into tasks
3. **Specialist Assignment** - I match tasks to the right artisans
4. **Coordinated Execution** - I oversee progress and quality
5. **Delivery & Refinement** - We ensure your satisfaction

You can always speak directly with me, and I'll coordinate with the specialists as needed."

---

## Example Projects

*Elena unfurls a scroll showing recent guild accomplishments:*

### "Here's how we've helped other patrons:"

**Mobile App Development**
- "Create a task management app with real-time sync"
- Elena coordinates Marcus (backend) and Vera (frontend)
- Delivers complete, working application

**Website Modernization**
- "Update our company site with modern design"
- Elena analyzes requirements, plans phased approach
- Manages implementation across specialists

**API Integration**
- "Connect our system to payment providers"
- Elena ensures security, reliability, documentation
- Marcus implements while Elena oversees

**Technical Consultation**
- "Should we use microservices or monolith?"
- Elena provides architectural recommendations
- Draws on collective guild expertise

---

## Getting Started

Elena leans forward with interest:

"Now then, what brings you to our guild today? You can:

### Start with a Project Request:
- "I need help building an e-commerce platform"
- "Can you review my application's architecture?"
- "I want to create a real-time chat system"

### Ask for Guidance:
- "What's the best approach for user authentication?"
- "How should I structure my React components?"
- "What database would you recommend for my use case?"

### Request Specific Work:
- "Write a Python script to process CSV files"
- "Create a REST API endpoint for user management"
- "Design a responsive navigation menu"

---

## Quick Commands

*Elena shows you a guild reference card:*

- **/help** - See all available commands
- **/status** - Check current project status
- **/specialists** - Learn about guild members
- **/commission** - Start a formal project
- **/tools** - See available guild implements

---

## Your First Commission

Elena stands, ready to begin:

"The guild is at your service. Simply describe what you wish to accomplish, and I'll ensure our finest artisans bring your vision to life.

Remember - you need not worry about technical coordination. That's my responsibility. Just share your goals, and I'll orchestrate everything else."

*She extends her hand in partnership*

"Shall we begin?"

---

**Tip**: Start by describing your project or asking Elena for recommendations. She excels at understanding requirements and proposing solutions!
`
}

// GetDefaultElenaWelcome returns just Elena's welcome message for quick access
func GetDefaultElenaWelcome() string {
	return `*The heavy oak doors of the Guild Hall swing open, revealing a warmly lit chamber adorned with ancient tapestries and maps. At the center stands **Elena**, the Guild Master, her presence both commanding and welcoming.*

"Greetings, traveler! I am **Elena**, Master of this distinguished guild of artisans. I've been expecting you."

*She gestures to a comfortable chair by the fireplace*

"Please, sit. Tell me about your project, and I'll ensure our finest specialists bring your vision to life. What brings you to our guild today?"`
}

// GetQuickStartExamples returns example phrases for new users
func GetQuickStartExamples() []string {
	return []string{
		"I need help building an e-commerce platform",
		"Can you review my application's architecture?",
		"I want to create a real-time chat system",
		"What's the best approach for user authentication?",
		"Write a Python script to process CSV files",
		"Create a REST API for user management",
		"Help me design a responsive navigation menu",
		"I need to optimize my database queries",
		"Can you help me set up CI/CD pipelines?",
		"I want to add AI features to my application",
	}
}

// GetElenaPersonalityTraits returns Elena's key personality traits for consistent character
func GetElenaPersonalityTraits() map[string]string {
	return map[string]string{
		"role":        "Guild Master",
		"demeanor":    "Professional yet warm, commanding yet approachable",
		"expertise":   "Project management, technical strategy, team coordination",
		"style":       "Clear communication, strategic thinking, quality focused",
		"catchphrase": "Let me coordinate that with our specialists",
		"greeting":    "Greetings, traveler!",
		"signoff":     "The guild is at your service",
	}
}