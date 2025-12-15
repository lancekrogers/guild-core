// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package campaign

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestDetectCampaign(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func(t *testing.T) string // returns test directory
		workingDir    string                    // relative to test directory
		explicitName  string
		wantCampaign  string
		wantErr       bool
		errorContains string
	}{
		{
			name: "detects campaign from local campaign.yaml",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				guildDir := filepath.Join(tmpDir, ".campaign")
				require.NoError(t, os.MkdirAll(guildDir, 0o755))

				// Create campaign reference
				refPath := filepath.Join(guildDir, "campaign.yaml")
				ref := CampaignReference{
					Campaign:    "e-commerce",
					Project:     "frontend",
					Description: "E-commerce frontend project",
				}
				data, err := yaml.Marshal(ref)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(refPath, data, 0o644))

				return tmpDir
			},
			workingDir:   ".",
			wantCampaign: "e-commerce",
		},
		{
			name: "detects campaign from parent directory campaign.yaml",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				guildDir := filepath.Join(tmpDir, ".campaign")
				require.NoError(t, os.MkdirAll(guildDir, 0o755))

				// Create campaign reference in parent
				refPath := filepath.Join(guildDir, "campaign.yaml")
				ref := CampaignReference{
					Campaign:    "task-manager",
					Project:     "api",
					Description: "Task manager API",
				}
				data, err := yaml.Marshal(ref)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(refPath, data, 0o644))

				// Create subdirectory
				subDir := filepath.Join(tmpDir, "src", "components")
				require.NoError(t, os.MkdirAll(subDir, 0o755))

				return tmpDir
			},
			workingDir:   "src/components",
			wantCampaign: "task-manager",
		},
		{
			name: "uses explicit campaign name when provided",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				guildDir := filepath.Join(tmpDir, ".campaign")
				require.NoError(t, os.MkdirAll(guildDir, 0o755))

				// Create campaign reference that should be ignored
				refPath := filepath.Join(guildDir, "campaign.yaml")
				ref := CampaignReference{
					Campaign:    "ignored-campaign",
					Project:     "test",
					Description: "Should be ignored",
				}
				data, err := yaml.Marshal(ref)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(refPath, data, 0o644))

				return tmpDir
			},
			workingDir:   ".",
			explicitName: "my-explicit-campaign",
			wantCampaign: "my-explicit-campaign",
		},
		{
			name: "returns error when no campaign detected",
			setupFunc: func(t *testing.T) string {
				// Create a directory with no .guild folder
				tmpDir := t.TempDir()
				return tmpDir
			},
			workingDir:    ".",
			wantErr:       true,
			errorContains: "no campaign found",
		},
		{
			name: "handles invalid campaign.yaml gracefully",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				guildDir := filepath.Join(tmpDir, ".campaign")
				require.NoError(t, os.MkdirAll(guildDir, 0o755))

				// Create invalid campaign.yaml
				refPath := filepath.Join(guildDir, "campaign.yaml")
				require.NoError(t, os.WriteFile(refPath, []byte("invalid yaml content {"), 0o644))

				return tmpDir
			},
			workingDir:    ".",
			wantErr:       true,
			errorContains: "no campaign found",
		},
		{
			name: "handles missing campaign name in campaign.yaml",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				guildDir := filepath.Join(tmpDir, ".campaign")
				require.NoError(t, os.MkdirAll(guildDir, 0o755))

				// Create campaign.yaml without campaign name
				refPath := filepath.Join(guildDir, "campaign.yaml")
				ref := CampaignReference{
					Project:     "test",
					Description: "Missing campaign name",
				}
				data, err := yaml.Marshal(ref)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(refPath, data, 0o644))

				return tmpDir
			},
			workingDir:    ".",
			wantErr:       true,
			errorContains: "no campaign found",
		},
		{
			name: "handles missing .guild directory in traversal",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				// Create deep directory structure without .guild
				deepDir := filepath.Join(tmpDir, "a", "b", "c", "d", "e")
				require.NoError(t, os.MkdirAll(deepDir, 0o755))
				return tmpDir
			},
			workingDir:    "a/b/c/d/e",
			wantErr:       true,
			errorContains: "no campaign found",
		},
		{
			name: "stops at filesystem root",
			setupFunc: func(t *testing.T) string {
				// This test verifies we don't panic when reaching root
				tmpDir := t.TempDir()
				return tmpDir
			},
			workingDir:    ".",
			wantErr:       true,
			errorContains: "no campaign found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up test directory
			testDir := tt.setupFunc(t)

			// Change to working directory
			fullWorkingDir := filepath.Join(testDir, tt.workingDir)

			// Call DetectCampaign
			gotCampaign, err := DetectCampaign(fullWorkingDir, tt.explicitName)

			// Check error
			if tt.wantErr {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantCampaign, gotCampaign)
		})
	}
}
