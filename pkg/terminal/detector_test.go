package terminal

import (
	"context"
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDetector(t *testing.T) {
	detector := NewDetector()
	assert.NotNil(t, detector)
	assert.NotEmpty(t, detector.platform)
	assert.Equal(t, runtime.GOOS, detector.platform)
}

func TestDetector_Detect(t *testing.T) {
	tests := []struct {
		name    string
		setup   func()
		cleanup func()
		wantErr bool
		check   func(*testing.T, Capabilities)
	}{
		{
			name: "basic detection",
			setup: func() {
				// Default environment
			},
			cleanup: func() {},
			wantErr: false,
			check: func(t *testing.T, caps Capabilities) {
				// Should have at least basic capabilities
				assert.True(t, caps.Colors >= NoColor)
				assert.NotNil(t, caps.Unicode)
				assert.NotNil(t, caps.Mouse)
			},
		},
		{
			name: "force color mode",
			setup: func() {
				os.Setenv("GUILD_FORCE_COLOR", "1")
			},
			cleanup: func() {
				os.Unsetenv("GUILD_FORCE_COLOR")
			},
			wantErr: false,
			check: func(t *testing.T, caps Capabilities) {
				assert.True(t, caps.Colors > NoColor)
			},
		},
		{
			name: "no color mode",
			setup: func() {
				os.Setenv("NO_COLOR", "1")
			},
			cleanup: func() {
				os.Unsetenv("NO_COLOR")
			},
			wantErr: false,
			check: func(t *testing.T, caps Capabilities) {
				assert.Equal(t, NoColor, caps.Colors)
			},
		},
		{
			name: "CI environment",
			setup: func() {
				os.Setenv("CI", "true")
			},
			cleanup: func() {
				os.Unsetenv("CI")
			},
			wantErr: false,
			check: func(t *testing.T, caps Capabilities) {
				// CI often has limited capabilities
				assert.False(t, caps.Mouse)
			},
		},
		{
			name: "dumb terminal",
			setup: func() {
				os.Setenv("TERM", "dumb")
			},
			cleanup: func() {
				os.Unsetenv("TERM")
			},
			wantErr: false,
			check: func(t *testing.T, caps Capabilities) {
				assert.Equal(t, NoColor, caps.Colors)
				assert.False(t, caps.Unicode)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			defer tt.cleanup()

			detector := NewDetector()
			ctx := context.Background()
			caps, err := detector.Detect(ctx)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				tt.check(t, caps)
			}
		})
	}
}

func TestDetector_detectColors(t *testing.T) {
	tests := []struct {
		name      string
		setupEnv  map[string]string
		want      ColorSupport
		wantTrueColor bool
	}{
		{
			name:      "no color",
			setupEnv:  map[string]string{"NO_COLOR": "1"},
			want:      NoColor,
			wantTrueColor: false,
		},
		{
			name:      "force color",
			setupEnv:  map[string]string{"GUILD_FORCE_COLOR": "1"},
			want:      Basic16,
			wantTrueColor: false,
		},
		{
			name:      "force true color",
			setupEnv:  map[string]string{"GUILD_FORCE_TRUE_COLOR": "1"},
			want:      TrueColor24Bit,
			wantTrueColor: true,
		},
		{
			name:      "256 color terminal",
			setupEnv:  map[string]string{"TERM": "xterm-256color"},
			want:      Extended256,
			wantTrueColor: false,
		},
		{
			name:      "true color terminal",
			setupEnv:  map[string]string{"COLORTERM": "truecolor"},
			want:      TrueColor24Bit,
			wantTrueColor: true,
		},
		{
			name:      "dumb terminal",
			setupEnv:  map[string]string{"TERM": "dumb"},
			want:      NoColor,
			wantTrueColor: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore environment
			oldEnv := make(map[string]string)
			for k := range tt.setupEnv {
				oldEnv[k] = os.Getenv(k)
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

			detector := NewDetector()
			colorSupport := detector.detectColorSupport()
			
			assert.Equal(t, tt.want, colorSupport)
			assert.Equal(t, tt.wantTrueColor, colorSupport == TrueColor24Bit)
		})
	}
}

func TestDetector_isCI(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected bool
	}{
		{
			name:     "GitHub Actions",
			envVars:  map[string]string{"GITHUB_ACTIONS": "true"},
			expected: true,
		},
		{
			name:     "GitLab CI",
			envVars:  map[string]string{"GITLAB_CI": "true"},
			expected: true,
		},
		{
			name:     "Jenkins",
			envVars:  map[string]string{"JENKINS_URL": "http://jenkins.example.com"},
			expected: true,
		},
		{
			name:     "Travis CI",
			envVars:  map[string]string{"TRAVIS": "true"},
			expected: true,
		},
		{
			name:     "CircleCI",
			envVars:  map[string]string{"CIRCLECI": "true"},
			expected: true,
		},
		{
			name:     "Generic CI",
			envVars:  map[string]string{"CI": "true"},
			expected: true,
		},
		{
			name:     "Not CI",
			envVars:  map[string]string{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore environment
			oldEnv := make(map[string]string)
			for k := range tt.envVars {
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
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			result := isRunningInCI()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetector_ConcurrentDetection(t *testing.T) {
	detector := NewDetector()
	ctx := context.Background()

	// Run multiple detections concurrently
	done := make(chan bool)
	results := make(chan Capabilities, 10)

	for i := 0; i < 10; i++ {
		go func() {
			caps, err := detector.Detect(ctx)
			if err == nil {
				results <- caps
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
	close(results)

	// All results should be the same (cached)
	var firstCaps Capabilities
	count := 0
	for caps := range results {
		if count == 0 {
			firstCaps = caps
		} else {
			assert.Equal(t, firstCaps, caps)
		}
		count++
	}
	assert.Greater(t, count, 0)
}

func TestDetector_ContextCancellation(t *testing.T) {
	detector := NewDetector()
	
	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	caps, err := detector.Detect(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context cancelled")
	assert.Equal(t, Capabilities{}, caps)
}

func TestDetector_EnvironmentOverrides(t *testing.T) {
	tests := []struct {
		name     string
		envVar   string
		value    string
		check    func(*testing.T, Capabilities)
	}{
		{
			name:   "force unicode",
			envVar: "GUILD_FORCE_UNICODE",
			value:  "1",
			check: func(t *testing.T, caps Capabilities) {
				assert.True(t, caps.Unicode)
			},
		},
		{
			name:   "force no unicode",
			envVar: "GUILD_FORCE_NO_UNICODE",
			value:  "1",
			check: func(t *testing.T, caps Capabilities) {
				assert.False(t, caps.Unicode)
			},
		},
		{
			name:   "force mouse",
			envVar: "GUILD_FORCE_MOUSE",
			value:  "1",
			check: func(t *testing.T, caps Capabilities) {
				assert.True(t, caps.Mouse)
			},
		},
		{
			name:   "force no mouse",
			envVar: "GUILD_FORCE_NO_MOUSE",
			value:  "1",
			check: func(t *testing.T, caps Capabilities) {
				assert.False(t, caps.Mouse)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldVal := os.Getenv(tt.envVar)
			os.Setenv(tt.envVar, tt.value)
			defer func() {
				if oldVal == "" {
					os.Unsetenv(tt.envVar)
				} else {
					os.Setenv(tt.envVar, oldVal)
				}
			}()

			detector := NewDetector()
			ctx := context.Background()
			caps, err := detector.Detect(ctx)
			
			require.NoError(t, err)
			tt.check(t, caps)
		})
	}
}