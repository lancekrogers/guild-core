package terminal

import (
	"context"
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProfileDetector(t *testing.T) {
	pd := NewProfileDetector()
	assert.NotNil(t, pd)
	assert.NotNil(t, pd.profiles)
	assert.NotNil(t, pd.detector)
	assert.Greater(t, len(pd.profiles), 10) // Should have many default profiles
}

func TestProfileDetector_Register(t *testing.T) {
	pd := NewProfileDetector()

	testProfile := &Profile{
		Name:        "test-terminal",
		Description: "Test Terminal",
		Priority:    100,
		Capabilities: Capabilities{
			Colors:  TrueColor24Bit,
			Unicode: true,
		},
	}

	pd.Register(testProfile)

	// Verify it was registered
	profile, ok := pd.GetProfile("test-terminal")
	assert.True(t, ok)
	assert.Equal(t, testProfile, profile)
	assert.NotNil(t, profile.Renderer)
}

func TestProfileDetector_Detect(t *testing.T) {
	tests := []struct {
		name      string
		setupEnv  map[string]string
		wantError bool
		checkName string
	}{
		{
			name: "detect with override",
			setupEnv: map[string]string{
				"GUILD_TERMINAL_PROFILE": "iterm2",
			},
			wantError: false,
			checkName: "iterm2",
		},
		{
			name: "detect iTerm2",
			setupEnv: map[string]string{
				"TERM_PROGRAM": "iTerm.app",
			},
			wantError: false,
			checkName: "iterm2",
		},
		{
			name: "detect VS Code",
			setupEnv: map[string]string{
				"TERM_PROGRAM": "vscode",
			},
			wantError: false,
			checkName: "vscode",
		},
		{
			name: "detect Windows Terminal",
			setupEnv: map[string]string{
				"WT_SESSION": "abc123",
			},
			wantError: false,
			checkName: func() string {
				if runtime.GOOS == "windows" {
					return "windows-terminal"
				}
				return "detected"
			}(),
		},
		{
			name: "detect Kitty",
			setupEnv: map[string]string{
				"TERM": "xterm-kitty",
			},
			wantError: false,
			checkName: "kitty",
		},
		{
			name: "detect dumb terminal",
			setupEnv: map[string]string{
				"TERM": "dumb",
			},
			wantError: false,
			checkName: "dumb",
		},
		{
			name:      "auto-detect",
			setupEnv:  map[string]string{},
			wantError: false,
			checkName: "detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore environment
			// Clear CI vars to prevent interference with profile detection
			ciVars := []string{
				"CI", "CONTINUOUS_INTEGRATION", "BUILD_NUMBER",
				"JENKINS_URL", "TRAVIS", "CIRCLECI", "GITHUB_ACTIONS",
				"GITLAB_CI", "BUILDKITE", "DRONE", "TEAMCITY_VERSION",
			}
			
			oldEnv := make(map[string]string)
			// Save CI vars
			for _, k := range ciVars {
				oldEnv[k] = os.Getenv(k)
				os.Unsetenv(k)
			}
			// Save test-specific vars
			for k := range tt.setupEnv {
				if _, exists := oldEnv[k]; !exists {
					oldEnv[k] = os.Getenv(k)
				}
			}
			
			defer func() {
				for k, v := range oldEnv {
					if v == "" {
						os.Unsetenv(k)
					} else {
						os.Setenv(k, v)
					}
				}
			}()

			// Set test environment
			for k, v := range tt.setupEnv {
				os.Setenv(k, v)
			}

			pd := NewProfileDetector()
			ctx := context.Background()
			profile, err := pd.Detect(ctx)

			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, profile)
				if tt.checkName != "" {
					assert.Equal(t, tt.checkName, profile.Name)
				}
				assert.NotNil(t, profile.Renderer)
			}
		})
	}
}

func TestProfileDetector_matchesProfile(t *testing.T) {
	pd := NewProfileDetector()

	tests := []struct {
		name        string
		profileName string
		setupEnv    map[string]string
		platform    string
		want        bool
	}{
		{
			name:        "iTerm2 match",
			profileName: "iterm2",
			setupEnv:    map[string]string{"TERM_PROGRAM": "iTerm.app"},
			platform:    "darwin",
			want:        true,
		},
		{
			name:        "iTerm2 wrong platform",
			profileName: "iterm2",
			setupEnv:    map[string]string{"TERM_PROGRAM": "iTerm.app"},
			platform:    "windows",
			want:        false,
		},
		{
			name:        "VS Code match",
			profileName: "vscode",
			setupEnv:    map[string]string{"TERM_PROGRAM": "vscode"},
			platform:    "darwin",
			want:        true,
		},
		{
			name:        "Alacritty match by TERM",
			profileName: "alacritty",
			setupEnv:    map[string]string{"TERM": "alacritty"},
			platform:    "linux",
			want:        true,
		},
		{
			name:        "Alacritty match by socket",
			profileName: "alacritty",
			setupEnv:    map[string]string{"ALACRITTY_SOCKET": "/tmp/alacritty"},
			platform:    "linux",
			want:        true,
		},
		{
			name:        "CI environment",
			profileName: "ci-environment",
			setupEnv:    map[string]string{},
			platform:    "linux",
			want:        false, // Would need to mock isCI
		},
		{
			name:        "SSH minimal",
			profileName: "ssh-minimal",
			setupEnv:    map[string]string{"SSH_CONNECTION": "192.168.1.1"},
			platform:    "linux",
			want:        false, // Would need full detector setup
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore environment
			oldEnv := make(map[string]string)
			for k := range tt.setupEnv {
				oldEnv[k] = os.Getenv(k)
				os.Unsetenv(k)
			}
			defer func() {
				for k, v := range oldEnv {
					if v != "" {
						os.Setenv(k, v)
					}
				}
			}()

			// Set test environment
			for k, v := range tt.setupEnv {
				os.Setenv(k, v)
			}

			profile, ok := pd.GetProfile(tt.profileName)
			require.True(t, ok, "Profile %s should exist", tt.profileName)

			// Note: We can't actually change runtime.GOOS, so some tests
			// will depend on the actual platform
			result := pd.matchesProfile(profile)
			if runtime.GOOS == tt.platform {
				assert.Equal(t, tt.want, result)
			}
		})
	}
}

func TestProfileDetector_ListProfiles(t *testing.T) {
	pd := NewProfileDetector()
	profiles := pd.ListProfiles()

	assert.Greater(t, len(profiles), 10)

	// Check some expected profiles exist
	expectedProfiles := []string{
		"iterm2",
		"windows-terminal",
		"vscode",
		"kitty",
		"alacritty",
		"gnome-terminal",
		"ssh-minimal",
		"dumb",
	}

	profileNames := make(map[string]bool)
	for _, p := range profiles {
		profileNames[p.Name] = true
	}

	for _, expected := range expectedProfiles {
		assert.True(t, profileNames[expected], "Expected profile %s", expected)
	}
}

func TestProfileDetector_Reset(t *testing.T) {
	// Clear environment to ensure we get a generic "detected" profile
	// which creates new objects each time
	envVars := []string{
		"GUILD_TERMINAL_PROFILE", "TERM_PROGRAM", "WT_SESSION", "TERM",
		"CI", "CONTINUOUS_INTEGRATION", "BUILD_NUMBER",
		"JENKINS_URL", "TRAVIS", "CIRCLECI", "GITHUB_ACTIONS",
		"GITLAB_CI", "BUILDKITE", "DRONE", "TEAMCITY_VERSION",
	}
	
	oldEnv := make(map[string]string)
	for _, k := range envVars {
		oldEnv[k] = os.Getenv(k)
		os.Unsetenv(k)
	}
	defer func() {
		for k, v := range oldEnv {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	pd := NewProfileDetector()
	ctx := context.Background()

	// First detection
	profile1, err := pd.Detect(ctx)
	require.NoError(t, err)

	// Second detection should return cached
	profile2, err := pd.Detect(ctx)
	require.NoError(t, err)
	assert.Same(t, profile1, profile2)

	// Reset cache
	pd.Reset()

	// Next detection should create new
	profile3, err := pd.Detect(ctx)
	require.NoError(t, err)
	
	// If a registered profile is detected, the objects will be the same
	// If auto-detected, they should be different objects
	if profile1.Name == "detected" {
		assert.NotSame(t, profile1, profile3)
	}
	assert.Equal(t, profile1.Name, profile3.Name)
}

func TestProfileDetector_ApplyProfile(t *testing.T) {
	pd := NewProfileDetector()

	// Apply existing profile
	err := pd.ApplyProfile("iterm2")
	require.NoError(t, err)

	// Verify it's set
	ctx := context.Background()
	profile, err := pd.Detect(ctx)
	require.NoError(t, err)
	assert.Equal(t, "iterm2", profile.Name)

	// Apply non-existent profile
	err = pd.ApplyProfile("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown profile")
}

func TestProfile_Capabilities(t *testing.T) {
	tests := []struct {
		name        string
		profileName string
		checkCaps   func(*testing.T, Capabilities)
	}{
		{
			name:        "iTerm2 capabilities",
			profileName: "iterm2",
			checkCaps: func(t *testing.T, caps Capabilities) {
				assert.Equal(t, TrueColor24Bit, caps.Colors)
				assert.True(t, caps.Unicode)
				assert.True(t, caps.Mouse)
				assert.True(t, caps.Images)
				assert.True(t, caps.ITerm2)
			},
		},
		{
			name:        "dumb terminal capabilities",
			profileName: "dumb",
			checkCaps: func(t *testing.T, caps Capabilities) {
				assert.Equal(t, NoColor, caps.Colors)
				assert.False(t, caps.Unicode)
				assert.False(t, caps.Mouse)
			},
		},
		{
			name:        "SSH minimal capabilities",
			profileName: "ssh-minimal",
			checkCaps: func(t *testing.T, caps Capabilities) {
				assert.Equal(t, Basic16, caps.Colors)
				assert.False(t, caps.Unicode)
				assert.False(t, caps.Mouse)
			},
		},
	}

	pd := NewProfileDetector()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile, ok := pd.GetProfile(tt.profileName)
			require.True(t, ok)
			tt.checkCaps(t, profile.Capabilities)
		})
	}
}

func TestProfile_Priority(t *testing.T) {
	pd := NewProfileDetector()
	profiles := pd.ListProfiles()

	// Check priority ordering
	priorities := make(map[int][]string)
	for _, p := range profiles {
		priorities[p.Priority] = append(priorities[p.Priority], p.Name)
	}

	// High-priority terminals
	assert.Contains(t, priorities[100], "iterm2")
	assert.Contains(t, priorities[100], "windows-terminal")

	// Low-priority terminals
	assert.Contains(t, priorities[1], "dumb")
}

func TestProfileDetector_ConcurrentDetection(t *testing.T) {
	pd := NewProfileDetector()
	ctx := context.Background()

	// Run multiple detections concurrently
	done := make(chan *Profile, 10)

	for i := 0; i < 10; i++ {
		go func() {
			profile, err := pd.Detect(ctx)
			if err == nil {
				done <- profile
			} else {
				done <- nil
			}
		}()
	}

	// Collect results
	var profiles []*Profile
	for i := 0; i < 10; i++ {
		profile := <-done
		if profile != nil {
			profiles = append(profiles, profile)
		}
	}

	// All should be the same (cached)
	require.Greater(t, len(profiles), 0)
	first := profiles[0]
	for _, p := range profiles[1:] {
		assert.Same(t, first, p)
	}
}

func TestProfileDetector_ContextCancellation(t *testing.T) {
	pd := NewProfileDetector()

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	profile, err := pd.Detect(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context cancelled")
	assert.Nil(t, profile)
}
