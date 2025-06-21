// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package chat

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// setupTestProjectContext creates .guild directory to make path appear as a project
func setupTestProjectContext(t *testing.T, tempDir string) {
	// Create .guild directory to make it a valid project
	guildDir := filepath.Join(tempDir, ".campaign")
	if err := os.MkdirAll(guildDir, 0755); err != nil {
		t.Fatalf("Failed to create .guild directory: %v", err)
	}
	
	// Change to the temp directory so GetContext() finds it
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	
	// Restore working directory after test
	t.Cleanup(func() {
		os.Chdir(originalWd)
	})
}

func TestNewGuildSelector(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "guild-selector-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	ctx := context.Background()
	setupTestProjectContext(t, tempDir)

	tests := []struct {
		name    string
		setup   func() error
		wantErr bool
		verify  func(t *testing.T, m *GuildSelectorModel)
	}{
		{
			name: "with existing guilds",
			setup: func() error {
				guildDir := filepath.Join(tempDir, ".campaign")
				os.MkdirAll(guildDir, 0755)
				
				// Create guild config
				guilds := &config.GuildConfigFile{
					Guilds: map[string]config.GuildDefinition{
						"backend-guild": {
							Purpose:     "Backend development",
							Description: "Handle backend tasks",
							Agents:      []string{"agent1", "agent2"},
						},
						"frontend-guild": {
							Purpose:     "Frontend development",
							Description: "Handle frontend tasks",
							Agents:      []string{"agent3"},
						},
					},
				}
				return config.SaveGuildConfigFile(ctx, tempDir, guilds)
			},
			wantErr: false,
			verify: func(t *testing.T, m *GuildSelectorModel) {
				if len(m.guilds) != 2 {
					t.Errorf("Expected 2 guilds, got %d", len(m.guilds))
				}
				if m.guilds[0].Name != "backend-guild" && m.guilds[0].Name != "frontend-guild" {
					t.Error("Guild names not loaded correctly")
				}
			},
		},
		{
			name: "with no guilds",
			setup: func() error {
				// Just create the .guild directory, no config files
				return os.MkdirAll(filepath.Join(tempDir, ".campaign"), 0755)
			},
			wantErr: false,
			verify: func(t *testing.T, m *GuildSelectorModel) {
				if len(m.guilds) != 0 {
					t.Errorf("Expected 0 guilds, got %d", len(m.guilds))
				}
			},
		},
		{
			name: "with campaign config and last selected",
			setup: func() error {
				guildDir := filepath.Join(tempDir, ".campaign")
				os.MkdirAll(guildDir, 0755)
				
				// Create guild config
				guilds := &config.GuildConfigFile{
					Guilds: map[string]config.GuildDefinition{
						"guild1": {
							Purpose:     "First guild",
							Description: "First",
							Agents:      []string{"agent1"},
						},
						"guild2": {
							Purpose:     "Second guild",
							Description: "Second",
							Agents:      []string{"agent2"},
						},
					},
				}
				config.SaveGuildConfigFile(ctx, tempDir, guilds)
				
				// Create campaign config with last selected
				campaign := &config.CampaignConfig{
					Name:              "test-campaign",
					Description:       "Test",
					LastSelectedGuild: "guild2",
				}
				return config.SaveCampaignConfig(ctx, tempDir, campaign)
			},
			wantErr: false,
			verify: func(t *testing.T, m *GuildSelectorModel) {
				if m.lastSelected != "guild2" {
					t.Errorf("Expected lastSelected='guild2', got '%s'", m.lastSelected)
				}
				// Check cursor is set to guild2
				for i, guild := range m.guilds {
					if guild.Name == "guild2" && m.cursor != i {
						t.Errorf("Cursor not set to last selected guild")
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up before each test
			os.RemoveAll(filepath.Join(tempDir, ".campaign"))
			
			if err := tt.setup(); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			m, err := NewGuildSelector(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewGuildSelector() error = %v, wantErr %v", err, tt.wantErr)
			}
			if m != nil && tt.verify != nil {
				tt.verify(t, m)
			}
		})
	}
}

func TestGuildSelectorModel_Init(t *testing.T) {
	m := &GuildSelectorModel{}
	cmd := m.Init()
	if cmd != nil {
		t.Error("Init() should return nil")
	}
}

func TestGuildSelectorModel_Update_Navigation(t *testing.T) {
	m := &GuildSelectorModel{
		guilds: []GuildInfo{
			{Name: "guild1", Description: "First"},
			{Name: "guild2", Description: "Second"},
			{Name: "guild3", Description: "Third"},
		},
		cursor: 0,
	}

	tests := []struct {
		name       string
		msg        tea.Msg
		wantCursor int
		wantQuit   bool
	}{
		{
			name:       "move down",
			msg:        tea.KeyMsg{Type: tea.KeyDown},
			wantCursor: 1,
		},
		{
			name:       "move down with j",
			msg:        tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}},
			wantCursor: 1,
		},
		{
			name:       "move up",
			msg:        tea.KeyMsg{Type: tea.KeyUp},
			wantCursor: 0, // Already at top
		},
		{
			name:       "move up with k",
			msg:        tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}},
			wantCursor: 0, // Already at top
		},
		{
			name:     "quit with q",
			msg:      tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}},
			wantQuit: true,
		},
		{
			name:     "quit with ctrl+c",
			msg:      tea.KeyMsg{Type: tea.KeyCtrlC},
			wantQuit: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset state
			m.cursor = 0
			m.quit = false

			_, _ = m.Update(tt.msg)

			if m.cursor != tt.wantCursor {
				t.Errorf("cursor = %d, want %d", m.cursor, tt.wantCursor)
			}
			if m.quit != tt.wantQuit {
				t.Errorf("quit = %v, want %v", m.quit, tt.wantQuit)
			}
		})
	}
}

func TestGuildSelectorModel_Update_Selection(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "guild-selector-select-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	ctx := context.Background()

	m := &GuildSelectorModel{
		guilds: []GuildInfo{
			{Name: "guild1", Description: "First"},
			{Name: "guild2", Description: "Second"},
		},
		cursor:      1,
		projectPath: tempDir,
		ctx:         ctx,
	}

	// Create .guild directory
	os.MkdirAll(filepath.Join(tempDir, ".campaign"), 0755)

	// Test selecting a guild
	model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	
	// Should have selected guild2
	if m.selected != "guild2" {
		t.Errorf("selected = %s, want guild2", m.selected)
	}
	
	// Should return saveSelection command
	if cmd == nil {
		t.Error("Expected saveSelection command")
	}

	// Execute the command to test save functionality
	msg := cmd()
	if _, ok := msg.(selectionSavedMsg); !ok {
		t.Error("Expected selectionSavedMsg")
	}

	// Update with the saved message
	_, cmd = model.Update(msg)
	if cmd == nil {
		t.Error("Expected quit command after save")
	}
}

func TestGuildSelectorModel_Update_CreateNew(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "guild-selector-create-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	ctx := context.Background()

	tests := []struct {
		name           string
		initialGuilds  []GuildInfo
		cursor         int
		keyMsg         tea.KeyMsg
		expectCreation bool
	}{
		{
			name:          "create with 'n' key",
			initialGuilds: []GuildInfo{{Name: "existing", Description: "Existing guild"}},
			cursor:        0,
			keyMsg:        tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}},
			expectCreation: true,
		},
		{
			name:           "create when no guilds exist",
			initialGuilds:  []GuildInfo{},
			cursor:         0,
			keyMsg:         tea.KeyMsg{Type: tea.KeyEnter},
			expectCreation: true,
		},
		{
			name: "create by selecting last option",
			initialGuilds: []GuildInfo{
				{Name: "guild1", Description: "First"},
				{Name: "guild2", Description: "Second"},
			},
			cursor:         2, // Cursor on "Create New Guild" option
			keyMsg:         tea.KeyMsg{Type: tea.KeyEnter},
			expectCreation: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up before each test
			os.RemoveAll(filepath.Join(tempDir, ".campaign"))
			os.MkdirAll(filepath.Join(tempDir, ".campaign"), 0755)

			m := &GuildSelectorModel{
				guilds:      tt.initialGuilds,
				cursor:      tt.cursor,
				projectPath: tempDir,
				ctx:         ctx,
				guildConfig: &config.GuildConfigFile{
					Guilds: make(map[string]config.GuildDefinition),
				},
			}

			// Create main guild.yaml to determine provider
			mainConfig := `name: test
providers:
  openai:
    settings: {}`
			os.WriteFile(filepath.Join(tempDir, ".campaign", "guild.yaml"), []byte(mainConfig), 0644)

			_, cmd := m.Update(tt.keyMsg)
			
			if tt.expectCreation {
				if cmd == nil {
					t.Error("Expected createDefaultGuild command")
				}
				
				// Execute the command
				msg := cmd()
				
				// Check for success
				if guildMsg, ok := msg.(guildCreatedMsg); ok {
					if !strings.Contains(string(guildMsg), "default-") {
						t.Errorf("Expected default guild name, got %s", string(guildMsg))
					}
				} else if errMsg, ok := msg.(errMsg); ok {
					t.Errorf("Creation failed with error: %v", errMsg.error)
				} else {
					t.Error("Unexpected message type from createDefaultGuild")
				}
			}
		})
	}
}

func TestGuildSelectorModel_Update_WindowSize(t *testing.T) {
	m := &GuildSelectorModel{}
	
	sizeMsg := tea.WindowSizeMsg{
		Width:  80,
		Height: 24,
	}
	
	m.Update(sizeMsg)
	
	if m.width != 80 {
		t.Errorf("width = %d, want 80", m.width)
	}
	if m.height != 24 {
		t.Errorf("height = %d, want 24", m.height)
	}
	if m.help.Width != 80 {
		t.Errorf("help.Width = %d, want 80", m.help.Width)
	}
}

func TestGuildSelectorModel_Update_ErrorHandling(t *testing.T) {
	m := &GuildSelectorModel{}
	
	testErr := gerror.New(gerror.ErrCodeInternal, "test error", nil)
	m.Update(errMsg{error: testErr})
	
	if m.err == nil {
		t.Error("Expected error to be set")
	}
	if m.err.Error() != testErr.Error() {
		t.Errorf("err = %v, want %v", m.err, testErr)
	}
}

func TestGuildSelectorModel_View(t *testing.T) {
	tests := []struct {
		name     string
		model    *GuildSelectorModel
		contains []string
	}{
		{
			name: "with guilds",
			model: &GuildSelectorModel{
				guilds: []GuildInfo{
					{Name: "backend", Description: "Backend guild", AgentCount: 3},
					{Name: "frontend", Description: "Frontend guild", AgentCount: 2},
				},
				cursor: 0,
			},
			contains: []string{
				"Select Guild",
				"backend (3 agents)",
				"frontend (2 agents)",
				"Create New Guild",
			},
		},
		{
			name: "no guilds",
			model: &GuildSelectorModel{
				guilds: []GuildInfo{},
				cursor: 0,
			},
			contains: []string{
				"No guilds found",
				"Create Default Guild",
			},
		},
		{
			name: "with error",
			model: &GuildSelectorModel{
				err: gerror.New(gerror.ErrCodeInternal, "test error", nil),
			},
			contains: []string{
				"Error:",
				"test error",
				"Press 'q' to quit",
			},
		},
		{
			name: "with last selected",
			model: &GuildSelectorModel{
				guilds: []GuildInfo{
					{Name: "guild1", Description: "First guild", AgentCount: 1},
					{Name: "guild2", Description: "Second guild", AgentCount: 2},
				},
				lastSelected: "guild2",
				cursor:       1,
			},
			contains: []string{
				"guild2 (2 agents)",
				"[last selected]",
			},
		},
		{
			name: "with help shown",
			model: &GuildSelectorModel{
				guilds:   []GuildInfo{{Name: "test", AgentCount: 1}},
				showHelp: true,
			},
			contains: []string{
				"↑/k",
				"↓/j",
				"enter",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := tt.model.View()
			
			for _, expected := range tt.contains {
				if !strings.Contains(view, expected) {
					t.Errorf("View() does not contain expected string: %s", expected)
				}
			}
		})
	}
}

func TestGuildSelectorModel_setCursorToGuild(t *testing.T) {
	m := &GuildSelectorModel{
		guilds: []GuildInfo{
			{Name: "alpha"},
			{Name: "beta"},
			{Name: "gamma"},
		},
		cursor: 0,
	}

	tests := []struct {
		name       string
		guildName  string
		wantCursor int
	}{
		{
			name:       "existing guild",
			guildName:  "beta",
			wantCursor: 1,
		},
		{
			name:       "non-existent guild",
			guildName:  "delta",
			wantCursor: 0, // Should remain at 0
		},
		{
			name:       "first guild",
			guildName:  "alpha",
			wantCursor: 0,
		},
		{
			name:       "last guild",
			guildName:  "gamma",
			wantCursor: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m.cursor = 0 // Reset cursor
			m.setCursorToGuild(tt.guildName)
			
			if m.cursor != tt.wantCursor {
				t.Errorf("cursor = %d, want %d", m.cursor, tt.wantCursor)
			}
		})
	}
}

func TestGuildSelectorModel_DefaultGuildCreation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "guild-default-create-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	ctx := context.Background()

	tests := []struct {
		name              string
		mainConfigContent string
		expectedProvider  string
		expectedPrefix    string
	}{
		{
			name: "claude provider",
			mainConfigContent: `name: test
providers:
  anthropic:
    settings: {}`,
			expectedProvider: "claude",
			expectedPrefix:   "claude",
		},
		{
			name: "ollama provider",
			mainConfigContent: `name: test
providers:
  ollama:
    base_url: http://localhost:11434`,
			expectedProvider: "ollama",
			expectedPrefix:   "local",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up before test
			os.RemoveAll(filepath.Join(tempDir, ".campaign"))
			os.MkdirAll(filepath.Join(tempDir, ".campaign"), 0755)

			// Write main config
			os.WriteFile(
				filepath.Join(tempDir, ".campaign", "guild.yaml"),
				[]byte(tt.mainConfigContent),
				0644,
			)

			m := &GuildSelectorModel{
				projectPath: tempDir,
				ctx:         ctx,
			}

			msg := m.createDefaultGuild()
			
			if guildMsg, ok := msg.(guildCreatedMsg); ok {
				expectedName := "default-" + tt.expectedProvider + "-guild"
				if string(guildMsg) != expectedName {
					t.Errorf("Expected guild name %s, got %s", expectedName, string(guildMsg))
				}
				
				// Verify guild was created
				guilds, err := config.LoadGuildConfigFile(ctx, tempDir)
				if err != nil {
					t.Errorf("Failed to load created guild config: %v", err)
				}
				
				guild, err := guilds.GetGuild(string(guildMsg))
				if err != nil {
					t.Errorf("Created guild not found: %v", err)
				}
				
				// Check agents
				expectedAgents := []string{
					tt.expectedPrefix + "-manager",
					tt.expectedPrefix + "-developer",
					tt.expectedPrefix + "-tester",
				}
				
				if len(guild.Agents) != len(expectedAgents) {
					t.Errorf("Expected %d agents, got %d", len(expectedAgents), len(guild.Agents))
				}
				
				for i, agent := range expectedAgents {
					if i < len(guild.Agents) && guild.Agents[i] != agent {
						t.Errorf("Expected agent %s, got %s", agent, guild.Agents[i])
					}
				}
			} else if errMsg, ok := msg.(errMsg); ok {
				t.Errorf("Failed to create default guild: %v", errMsg.error)
			}
		})
	}
}

func TestGuildSelectorModel_SaveSelection(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "guild-save-selection-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	ctx := context.Background()
	os.MkdirAll(filepath.Join(tempDir, ".campaign"), 0755)

	m := &GuildSelectorModel{
		projectPath: tempDir,
		ctx:         ctx,
		selected:    "test-guild",
	}

	msg := m.saveSelection()
	
	if _, ok := msg.(selectionSavedMsg); !ok {
		t.Error("Expected selectionSavedMsg")
	}
	
	// Verify campaign config was created/updated
	campaign, err := config.LoadCampaignConfig(ctx, tempDir)
	if err != nil {
		t.Errorf("Failed to load campaign config: %v", err)
	}
	
	if campaign.LastSelectedGuild != "test-guild" {
		t.Errorf("LastSelectedGuild = %s, want test-guild", campaign.LastSelectedGuild)
	}
}

func TestRunGuildSelector(t *testing.T) {
	// This test is limited because it requires a full terminal environment
	// We can test error cases
	
	t.Run("cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		
		_, err := RunGuildSelector(ctx)
		if err == nil {
			t.Error("Expected error with cancelled context")
		}
	})
	
	// Note: Full integration testing of RunGuildSelector would require
	// mocking the tea.Program or running in a PTY environment
}


// State transition tests
func TestGuildSelectorModel_StateTransitions(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "guild-state-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	ctx := context.Background()
	setupTestProjectContext(t, tempDir)

	// Create initial guild config
	guildDir := filepath.Join(tempDir, ".campaign")
	os.MkdirAll(guildDir, 0755)
	
	guilds := &config.GuildConfigFile{
		Guilds: map[string]config.GuildDefinition{
			"guild1": {
				Purpose:     "First",
				Description: "First guild",
				Agents:      []string{"agent1"},
			},
		},
	}
	config.SaveGuildConfigFile(ctx, tempDir, guilds)

	// Initialize selector
	m, err := NewGuildSelector(ctx)
	if err != nil {
		t.Fatalf("Failed to create selector: %v", err)
	}

	// Test state transitions
	transitions := []struct {
		name      string
		action    tea.Msg
		checkFunc func(t *testing.T)
	}{
		{
			name:   "initial state",
			action: nil,
			checkFunc: func(t *testing.T) {
				if m.cursor != 0 {
					t.Error("Initial cursor should be 0")
				}
				if m.selected != "" {
					t.Error("No guild should be selected initially")
				}
			},
		},
		{
			name:   "toggle help",
			action: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}},
			checkFunc: func(t *testing.T) {
				if !m.showHelp {
					t.Error("Help should be shown")
				}
			},
		},
		{
			name:   "move to create option",
			action: tea.KeyMsg{Type: tea.KeyDown},
			checkFunc: func(t *testing.T) {
				if m.cursor != 1 {
					t.Error("Cursor should be on create option")
				}
			},
		},
		{
			name:   "select guild",
			action: tea.KeyMsg{Type: tea.KeyUp},
			checkFunc: func(t *testing.T) {
				m.Update(tea.KeyMsg{Type: tea.KeyEnter})
				if m.selected != "guild1" {
					t.Errorf("Guild1 should be selected, got %s", m.selected)
				}
			},
		},
	}

	for _, tr := range transitions {
		t.Run(tr.name, func(t *testing.T) {
			if tr.action != nil {
				m.Update(tr.action)
			}
			tr.checkFunc(t)
		})
	}
}

// Edge case tests
func TestGuildSelectorModel_EdgeCases(t *testing.T) {
	t.Run("navigation with single guild", func(t *testing.T) {
		m := &GuildSelectorModel{
			guilds: []GuildInfo{
				{Name: "only-guild"},
			},
			cursor: 0,
		}
		
		// Try to move down - should go to create option
		m.Update(tea.KeyMsg{Type: tea.KeyDown})
		if m.cursor != 1 {
			t.Error("Should move to create option")
		}
		
		// Try to move down again - should stay
		m.Update(tea.KeyMsg{Type: tea.KeyDown})
		if m.cursor != 1 {
			t.Error("Should stay at create option")
		}
		
		// Move up
		m.Update(tea.KeyMsg{Type: tea.KeyUp})
		if m.cursor != 0 {
			t.Error("Should move back to guild")
		}
		
		// Try to move up again - should stay
		m.Update(tea.KeyMsg{Type: tea.KeyUp})
		if m.cursor != 0 {
			t.Error("Should stay at first guild")
		}
	})

	t.Run("view with very long guild names", func(t *testing.T) {
		m := &GuildSelectorModel{
			guilds: []GuildInfo{
				{
					Name:        "this-is-a-very-long-guild-name-that-might-cause-display-issues",
					Description: "This is an extremely long description that goes on and on and might need to be wrapped or truncated in the display",
					AgentCount:  99,
				},
			},
		}
		
		view := m.View()
		// Just ensure it doesn't panic
		if view == "" {
			t.Error("View should not be empty")
		}
	})

	t.Run("concurrent state updates", func(t *testing.T) {
		m := &GuildSelectorModel{
			guilds: make([]GuildInfo, 10),
		}
		
		// Fill with test data
		for i := 0; i < 10; i++ {
			m.guilds[i] = GuildInfo{
				Name:       string(rune('a' + i)),
				AgentCount: i,
			}
		}
		
		// Note: The actual GuildSelectorModel doesn't have concurrent
		// access in practice since Bubble Tea is single-threaded,
		// but this tests that our View method is safe
		done := make(chan bool, 2)
		
		go func() {
			for i := 0; i < 100; i++ {
				_ = m.View()
			}
			done <- true
		}()
		
		go func() {
			for i := 0; i < 100; i++ {
				m.cursor = i % 10
			}
			done <- true
		}()
		
		<-done
		<-done
	})
}