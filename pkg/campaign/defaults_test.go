// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package campaign

import (
	"strings"
	"testing"
)

func TestGetDefaultCampaign(t *testing.T) {
	campaign := GetDefaultCampaign()
	
	// Test basic properties
	if campaign.ID != "default-guild-welcome" {
		t.Errorf("Expected ID 'default-guild-welcome', got %s", campaign.ID)
	}
	
	if campaign.Name != "Guild Introduction" {
		t.Errorf("Expected Name 'Guild Introduction', got %s", campaign.Name)
	}
	
	if campaign.Status != CampaignStatusActive {
		t.Errorf("Expected Status %s, got %s", CampaignStatusActive, campaign.Status)
	}
	
	// Test that description contains Elena content
	if !strings.Contains(campaign.Description, "Elena") {
		t.Error("Campaign description should mention Elena")
	}
	
	if !strings.Contains(campaign.Description, "Guild Master") {
		t.Error("Campaign description should mention Guild Master")
	}
	
	// Test metadata
	if campaign.Metadata["manager"] != "elena" {
		t.Errorf("Expected manager metadata to be 'elena', got %v", campaign.Metadata["manager"])
	}
	
	// Test tags
	expectedTags := []string{"welcome", "tutorial", "elena", "guild"}
	if len(campaign.Tags) != len(expectedTags) {
		t.Errorf("Expected %d tags, got %d", len(expectedTags), len(campaign.Tags))
	}
}

func TestGetDefaultElenaWelcome(t *testing.T) {
	welcome := GetDefaultElenaWelcome()
	
	// Test key elements are present
	elements := []string{
		"heavy oak doors",
		"Elena",
		"Guild Master",
		"Greetings, traveler",
		"I've been expecting you",
		"Tell me about your project",
	}
	
	for _, element := range elements {
		if !strings.Contains(welcome, element) {
			t.Errorf("Welcome message missing expected element: %s", element)
		}
	}
}

func TestGetQuickStartExamples(t *testing.T) {
	examples := GetQuickStartExamples()
	
	if len(examples) == 0 {
		t.Error("Expected at least some quick start examples")
	}
	
	// Check a few specific examples
	expectedExamples := []string{
		"I need help building an e-commerce platform",
		"What's the best approach for user authentication?",
		"Create a REST API for user management",
	}
	
	for _, expected := range expectedExamples {
		found := false
		for _, example := range examples {
			if example == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected example not found: %s", expected)
		}
	}
}

func TestGetElenaPersonalityTraits(t *testing.T) {
	traits := GetElenaPersonalityTraits()
	
	// Test key personality traits
	expectedTraits := map[string]string{
		"role":        "Guild Master",
		"greeting":    "Greetings, traveler!",
		"signoff":     "The guild is at your service",
	}
	
	for key, expected := range expectedTraits {
		if actual, ok := traits[key]; !ok {
			t.Errorf("Missing personality trait: %s", key)
		} else if actual != expected {
			t.Errorf("Trait %s: expected '%s', got '%s'", key, expected, actual)
		}
	}
}