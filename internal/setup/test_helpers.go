// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package setup

import (
	"os"
	"path/filepath"
	"testing"

	yaml "gopkg.in/yaml.v3"

	"github.com/guild-ventures/guild-core/pkg/config"
)

// setupCampaignStructure creates the proper campaign-first directory structure for tests
func setupCampaignStructure(t *testing.T, projectPath string, guildConfig *config.GuildConfig) error {
	t.Helper()

	// Create directory structure
	dirs := []string{
		filepath.Join(projectPath, ".campaign"),
		filepath.Join(projectPath, ".campaign", "guilds"),
		filepath.Join(projectPath, ".campaign", "agents"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Create campaign.yaml
	campaignData := map[string]interface{}{
		"name":        "test-campaign",
		"description": "Test campaign for unit tests",
		"guilds":      []string{guildConfig.Name},
		"settings": map[string]interface{}{
			"default_guild": guildConfig.Name,
		},
	}
	campaignYAML, err := yaml.Marshal(campaignData)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(projectPath, ".campaign", "campaign.yaml"), campaignYAML, 0644); err != nil {
		return err
	}

	// Create guild definition
	guildDef := map[string]interface{}{
		"name":        guildConfig.Name,
		"description": guildConfig.Description,
		"purpose":     "Test guild for unit tests",
		"manager":     guildConfig.Manager.Default,
		"agents":      []string{},
	}
	// Collect agent IDs
	for _, agent := range guildConfig.Agents {
		guildDef["agents"] = append(guildDef["agents"].([]string), agent.ID)
	}
	guildYAML, err := yaml.Marshal(guildDef)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(projectPath, ".campaign", "guilds", guildConfig.Name+".yaml"), guildYAML, 0644); err != nil {
		return err
	}

	// Create agent files
	for _, agent := range guildConfig.Agents {
		agentYAML, err := yaml.Marshal(agent)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(projectPath, ".campaign", "agents", agent.ID+".yaml"), agentYAML, 0644); err != nil {
			return err
		}
	}

	return nil
}
