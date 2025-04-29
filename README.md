# Guild: A Golang AI-Agent Framework

## Overview

**Guild** is a Golang framework for building efficient, agentic workflows. It allows you to create customized teams of AI agents ("guilds") that can execute and coordinate tasks to automate projects and boost productivity.

Think of a **guild** as a team of agents, each assigned to a specific role. Guilds can be tailored for a single project or used at a higher level to oversee multiple projects.

## Example Usage

- Create one **guild per project**, using small, fine-tuned models for specific roles.
- Create an **overseer guild** using larger models (e.g., GPT-4) to review workflows from multiple projects or coordinate long-term planning.
- Project guilds can optionally submit completed tasks to a human-review queue or to other agents for QA or refinement.

A **project** is defined as a set of ordered tasks in a configuration file. Each task can contain **subtasks**, which must be completed before the parent task is marked complete. Agents can dynamically add or remove subtasks if they identify missing requirements.

Each guild member works on one task at a time, allowing clear tracking and scoped concurrency. Agents can add task to team ques if requirements aren't detailed enough to complete a task.

## Key Features

- Configurable projects and agents
- Multi-guild support with delegation
- Dynamic task ordering and dependency handling
- CLI dashboard to monitor status and task progress

## Installation

1. Clone the repo:

   ```bash
   git clone https://github.com/lancekrogers/guild.git
   cd guild
   ```
