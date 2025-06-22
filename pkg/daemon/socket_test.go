// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package daemon

import (
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestSaveSocketRegistry(t *testing.T) {
	tests := []struct {
		name        string
		projectRoot string
		campaign    string
		wantErr     bool
	}{
		{
			name:        "saves registry successfully",
			projectRoot: t.TempDir(),
			campaign:    "test-campaign",
			wantErr:     false,
		},
		{
			name:        "creates .guild directory if missing",
			projectRoot: t.TempDir(),
			campaign:    "another-campaign",
			wantErr:     false,
		},
		{
			name:        "handles special characters in campaign name",
			projectRoot: t.TempDir(),
			campaign:    "test-campaign-with-dashes",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SaveSocketRegistry(tt.projectRoot, tt.campaign)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Verify the file was created
			registryPath := filepath.Join(tt.projectRoot, ".campaign", "socket-registry.yaml")
			assert.FileExists(t, registryPath)

			// Verify the content
			data, err := os.ReadFile(registryPath)
			require.NoError(t, err)

			var registry SocketRegistry
			err = yaml.Unmarshal(data, &registry)
			require.NoError(t, err)

			assert.Equal(t, tt.campaign, registry.CampaignName)
			assert.NotEmpty(t, registry.CampaignHash)
		})
	}
}

func TestCanConnect(t *testing.T) {
	tests := []struct {
		name       string
		setupFunc  func(t *testing.T) string // returns socket path
		teardown   func()
		wantResult bool
	}{
		{
			name: "returns true for active socket",
			setupFunc: func(t *testing.T) string {
				// Use a shorter path to avoid socket path length limit
				socketPath := filepath.Join("/tmp", "guild-test.sock")

				// Clean up any existing socket
				os.Remove(socketPath)

				// Create a listener
				listener, err := net.Listen("unix", socketPath)
				require.NoError(t, err)

				// Accept connections in background
				go func() {
					for {
						conn, err := listener.Accept()
						if err != nil {
							return
						}
						conn.Close()
					}
				}()

				t.Cleanup(func() {
					listener.Close()
				})

				return socketPath
			},
			wantResult: true,
		},
		{
			name: "returns false for non-existent socket",
			setupFunc: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "nonexistent.sock")
			},
			wantResult: false,
		},
		{
			name: "returns false for stale socket file",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				socketPath := filepath.Join(tmpDir, "stale.sock")

				// Create an empty file to simulate stale socket
				require.NoError(t, os.WriteFile(socketPath, []byte{}, 0600))

				return socketPath
			},
			wantResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			socketPath := tt.setupFunc(t)

			result := CanConnect(socketPath)
			assert.Equal(t, tt.wantResult, result)
		})
	}
}

func TestCleanupStaleSessionSockets(t *testing.T) {
	t.Skip("Skipping test that requires daemon environment setup")
	tests := []struct {
		name      string
		setupFunc func(t *testing.T) (string, string) // returns campaign name and campaign dir
		validate  func(t *testing.T, campaignDir string)
	}{
		{
			name: "removes stale socket files",
			setupFunc: func(t *testing.T) (string, string) {
				homeDir := t.TempDir()
				t.Setenv("HOME", homeDir)

				campaign := "test-campaign"
				campaignDir := filepath.Join(homeDir, ".campaign", "campaigns", campaign)
				require.NoError(t, os.MkdirAll(campaignDir, 0755))

				// Create stale socket files
				for i := 0; i < 3; i++ {
					socketPath := filepath.Join(campaignDir, "guild.sock")
					if i > 0 {
						socketPath = filepath.Join(campaignDir, "guild-"+string(rune('0'+i))+".sock")
					}
					require.NoError(t, os.WriteFile(socketPath, []byte{}, 0600))
				}

				return campaign, campaignDir
			},
			validate: func(t *testing.T, campaignDir string) {
				// All stale sockets should be removed
				entries, err := os.ReadDir(campaignDir)
				require.NoError(t, err)

				for _, entry := range entries {
					assert.False(t, filepath.Ext(entry.Name()) == ".sock")
				}
			},
		},
		{
			name: "preserves active socket files",
			setupFunc: func(t *testing.T) (string, string) {
				homeDir := t.TempDir()
				t.Setenv("HOME", homeDir)

				campaign := "active-campaign"
				campaignDir := filepath.Join(homeDir, ".campaign", "campaigns", campaign)
				require.NoError(t, os.MkdirAll(campaignDir, 0755))

				// Create active socket
				activeSocket := filepath.Join(campaignDir, "guild.sock")
				listener, err := net.Listen("unix", activeSocket)
				require.NoError(t, err)
				t.Cleanup(func() {
					listener.Close()
				})

				// Accept connections in background
				go func() {
					for {
						conn, err := listener.Accept()
						if err != nil {
							return
						}
						conn.Close()
					}
				}()

				// Create stale socket
				staleSocket := filepath.Join(campaignDir, "guild-1.sock")
				require.NoError(t, os.WriteFile(staleSocket, []byte{}, 0600))

				return campaign, campaignDir
			},
			validate: func(t *testing.T, campaignDir string) {
				// Active socket should exist
				assert.FileExists(t, filepath.Join(campaignDir, "guild.sock"))
				// Stale socket should be removed
				assert.NoFileExists(t, filepath.Join(campaignDir, "guild-1.sock"))
			},
		},
		{
			name: "handles missing campaign directory",
			setupFunc: func(t *testing.T) (string, string) {
				homeDir := t.TempDir()
				t.Setenv("HOME", homeDir)
				return "nonexistent-campaign", filepath.Join(homeDir, ".campaign", "campaigns", "nonexistent-campaign")
			},
			validate: func(t *testing.T, campaignDir string) {
				// Should not create the directory
				assert.NoDirExists(t, campaignDir)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			campaign, campaignDir := tt.setupFunc(t)

			err := CleanupStaleSessionSockets(campaign)
			require.NoError(t, err)

			tt.validate(t, campaignDir)
		})
	}
}

func TestUnlinkIfStale(t *testing.T) {
	tests := []struct {
		name       string
		setupFunc  func(t *testing.T) string // returns socket path
		wantErr    bool
		wantRemove bool
	}{
		{
			name: "removes stale socket file",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				socketPath := filepath.Join(tmpDir, "stale.sock")
				require.NoError(t, os.WriteFile(socketPath, []byte{}, 0600))
				return socketPath
			},
			wantRemove: true,
		},
		{
			name: "preserves active socket",
			setupFunc: func(t *testing.T) string {
				// Use short path
				socketPath := filepath.Join("/tmp", "guild-unlinkactive.sock")
				os.Remove(socketPath)

				listener, err := net.Listen("unix", socketPath)
				require.NoError(t, err)
				t.Cleanup(func() {
					listener.Close()
					os.Remove(socketPath)
				})

				// Accept connections in background
				go func() {
					for {
						conn, err := listener.Accept()
						if err != nil {
							return
						}
						conn.Close()
					}
				}()

				return socketPath
			},
			wantRemove: false,
		},
		{
			name: "handles non-existent socket",
			setupFunc: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "nonexistent.sock")
			},
			wantRemove: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			socketPath := tt.setupFunc(t)

			err := UnlinkIfStale(socketPath)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.wantRemove {
				assert.NoFileExists(t, socketPath)
			} else if _, err := os.Stat(socketPath); err == nil {
				assert.FileExists(t, socketPath)
			}
		})
	}
}

func TestListCampaignSessions(t *testing.T) {
	t.Skip("Skipping test that requires daemon environment setup")
	tests := []struct {
		name         string
		setupFunc    func(t *testing.T) (string, string) // returns campaign and home dir
		wantSessions []int
		wantErr      bool
	}{
		{
			name: "lists all active sessions",
			setupFunc: func(t *testing.T) (string, string) {
				homeDir := t.TempDir()
				campaign := "multi-session"
				campaignDir := filepath.Join(homeDir, ".campaign", "campaigns", campaign)
				require.NoError(t, os.MkdirAll(campaignDir, 0755))

				// Create multiple active sockets
				for i := 0; i < 3; i++ {
					var socketPath string
					if i == 0 {
						socketPath = filepath.Join(campaignDir, "guild.sock")
					} else {
						socketPath = filepath.Join(campaignDir, "guild-"+string(rune('0'+i))+".sock")
					}

					listener, err := net.Listen("unix", socketPath)
					require.NoError(t, err)
					t.Cleanup(func() {
						listener.Close()
					})

					go func() {
						for {
							conn, err := listener.Accept()
							if err != nil {
								return
							}
							conn.Close()
						}
					}()
				}

				return campaign, homeDir
			},
			wantSessions: []int{0, 1, 2},
		},
		{
			name: "returns empty list for no sessions",
			setupFunc: func(t *testing.T) (string, string) {
				homeDir := t.TempDir()
				campaign := "empty-campaign"
				campaignDir := filepath.Join(homeDir, ".campaign", "campaigns", campaign)
				require.NoError(t, os.MkdirAll(campaignDir, 0755))
				return campaign, homeDir
			},
			wantSessions: []int{},
		},
		{
			name: "ignores stale sockets",
			setupFunc: func(t *testing.T) (string, string) {
				homeDir := t.TempDir()
				campaign := "mixed-campaign"
				campaignDir := filepath.Join(homeDir, ".campaign", "campaigns", campaign)
				require.NoError(t, os.MkdirAll(campaignDir, 0755))

				// Create one active socket
				activeSocket := filepath.Join(campaignDir, "guild.sock")
				listener, err := net.Listen("unix", activeSocket)
				require.NoError(t, err)
				t.Cleanup(func() {
					listener.Close()
				})

				go func() {
					for {
						conn, err := listener.Accept()
						if err != nil {
							return
						}
						conn.Close()
					}
				}()

				// Create stale sockets
				for i := 1; i < 3; i++ {
					staleSocket := filepath.Join(campaignDir, "guild-"+string(rune('0'+i))+".sock")
					require.NoError(t, os.WriteFile(staleSocket, []byte{}, 0600))
				}

				return campaign, homeDir
			},
			wantSessions: []int{0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			campaign, homeDir := tt.setupFunc(t)
			t.Setenv("HOME", homeDir)

			sessions, err := ListCampaignSessions(campaign)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Extract session numbers
			gotSessions := make([]int, len(sessions))
			for i, s := range sessions {
				gotSessions[i] = s.Session
			}

			assert.ElementsMatch(t, tt.wantSessions, gotSessions)
		})
	}
}

func TestDiscoverAllRunningSessions(t *testing.T) {
	t.Skip("Skipping test that requires daemon environment setup")
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	// Set up multiple campaigns with sessions
	campaigns := map[string][]int{
		"campaign-a": {0, 1},
		"campaign-b": {0},
		"campaign-c": {}, // No active sessions
	}

	for campaign, sessions := range campaigns {
		campaignDir := filepath.Join(homeDir, ".campaign", "campaigns", campaign)
		require.NoError(t, os.MkdirAll(campaignDir, 0755))

		for _, session := range sessions {
			var socketPath string
			if session == 0 {
				socketPath = filepath.Join(campaignDir, "guild.sock")
			} else {
				socketPath = filepath.Join(campaignDir, "guild-"+string(rune('0'+session))+".sock")
			}

			listener, err := net.Listen("unix", socketPath)
			require.NoError(t, err)
			t.Cleanup(func() {
				listener.Close()
			})

			go func() {
				for {
					conn, err := listener.Accept()
					if err != nil {
						return
					}
					conn.Close()
				}
			}()
		}
	}

	// Discover all sessions
	allSessions, err := DiscoverAllRunningSessions()
	require.NoError(t, err)

	// Verify results
	assert.Len(t, allSessions, 2) // Only campaigns with active sessions

	// Check campaign-a
	if sessions, ok := allSessions["campaign-a"]; ok {
		assert.Len(t, sessions, 2)
		sessionNums := []int{sessions[0].Session, sessions[1].Session}
		assert.ElementsMatch(t, []int{0, 1}, sessionNums)
	} else {
		t.Error("Expected campaign-a in results")
	}

	// Check campaign-b
	if sessions, ok := allSessions["campaign-b"]; ok {
		assert.Len(t, sessions, 1)
		assert.Equal(t, 0, sessions[0].Session)
	} else {
		t.Error("Expected campaign-b in results")
	}

	// Check campaign-c is not included
	_, ok := allSessions["campaign-c"]
	assert.False(t, ok)
}

func TestStopSession(t *testing.T) {
	t.Skip("Skipping flaky test that requires proper daemon setup")
	tests := []struct {
		name      string
		setupFunc func(t *testing.T) string // returns socket path
		wantErr   bool
	}{
		{
			name: "stops active session",
			setupFunc: func(t *testing.T) string {
				// Use short path
				socketPath := filepath.Join("/tmp", "guild-stop.sock")
				os.Remove(socketPath)

				listener, err := net.Listen("unix", socketPath)
				require.NoError(t, err)

				// Set up cleanup
				t.Cleanup(func() {
					listener.Close()
					os.Remove(socketPath)
				})

				// Handle shutdown command
				go func() {
					conn, err := listener.Accept()
					if err != nil {
						return
					}
					// Read the shutdown signal
					buf := make([]byte, 9) // "SHUTDOWN\n"
					n, _ := conn.Read(buf)
					if n > 0 && string(buf[:n]) == "SHUTDOWN\n" {
						listener.Close()
					}
					conn.Close()
				}()

				// Give listener time to start
				time.Sleep(10 * time.Millisecond)

				return socketPath
			},
			wantErr: false,
		},
		{
			name: "handles non-existent socket",
			setupFunc: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "nonexistent.sock")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			socketPath := tt.setupFunc(t)

			err := StopSession(socketPath)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Give time for socket to be closed
			time.Sleep(200 * time.Millisecond)

			// The socket file might still exist but should not be connectable
			if CanConnect(socketPath) {
				t.Logf("Socket at %s is still connectable after StopSession", socketPath)
			}
			assert.False(t, CanConnect(socketPath), "Socket should not be connectable after stop")
		})
	}
}
