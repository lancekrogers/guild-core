# Objective Creation Prompt

You are an AI assistant tasked with creating a well-structured objective document from a user's description.

## User Description

{{.Description}}

## Your Task

Create a markdown document that describes this objective with the following structure:

# 🧠 Goal

- Clear, concise statement of the main objective

# 📂 Context

- Background information necessary to understand this objective
- Any relevant history or previous work

# 🔧 Requirements

- Specific requirements that must be met
- Constraints or limitations to be aware of
- Key functionality that must be implemented

# 📌 Tags

- 3-5 tags that categorize this objective (comma-separated)

# 🔗 Related

- Links to related objectives or dependencies

Ensure your response follows this exact format and is complete enough to serve as a standalone document.
