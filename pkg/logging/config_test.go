package logging

import (
	"log/slog"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Level != slog.LevelInfo {
		t.Errorf("Expected info level, got %v", cfg.Level)
	}

	if cfg.Format != "json" {
		t.Errorf("Expected json format, got %s", cfg.Format)
	}

	if cfg.Development {
		t.Error("Expected production mode")
	}
}

func TestDevelopmentConfig(t *testing.T) {
	cfg := DevelopmentConfig()

	if cfg.Level != slog.LevelDebug {
		t.Errorf("Expected debug level, got %v", cfg.Level)
	}

	if cfg.Format != "pretty" {
		t.Errorf("Expected pretty format, got %s", cfg.Format)
	}

	if !cfg.Development {
		t.Error("Expected development mode")
	}
}

func TestProductionConfig(t *testing.T) {
	cfg := ProductionConfig("/var/log/app.log")

	if cfg.Level != slog.LevelInfo {
		t.Errorf("Expected info level, got %v", cfg.Level)
	}

	if cfg.Format != "json" {
		t.Errorf("Expected json format, got %s", cfg.Format)
	}

	if cfg.Development {
		t.Error("Expected production mode")
	}

	if !cfg.Sampling.Enabled {
		t.Error("Expected sampling enabled")
	}
	if cfg.Sampling.Type != "adaptive" {
		t.Errorf("Expected adaptive sampling type, got %s", cfg.Sampling.Type)
	}
	if cfg.Sampling.TargetRate != 1000 {
		t.Errorf("Expected target rate 1000, got %d", cfg.Sampling.TargetRate)
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: Config{
				Level:  slog.LevelInfo,
				Format: "json",
				Output: "stdout",
			},
			wantErr: false,
		},
		{
			name: "invalid format",
			cfg: Config{
				Level:  slog.LevelInfo,
				Format: "invalid",
			},
			wantErr: true,
		},
		{
			name:    "empty config uses defaults",
			cfg:     Config{},
			wantErr: true, // Empty config should fail validation
		},
		{
			name: "warn level",
			cfg: Config{
				Level:  slog.LevelWarn,
				Format: "json",
				Output: "stdout",
			},
			wantErr: false,
		},
		{
			name: "error level",
			cfg: Config{
				Level:  slog.LevelError,
				Format: "json",
				Output: "stdout",
			},
			wantErr: false,
		},
		{
			name: "text format",
			cfg: Config{
				Level:  slog.LevelInfo,
				Format: "text",
				Output: "stdout",
			},
			wantErr: false,
		},
		{
			name: "pretty format",
			cfg: Config{
				Level:  slog.LevelInfo,
				Format: "pretty",
				Output: "stdout",
			},
			wantErr: false,
		},
		{
			name: "with file path",
			cfg: Config{
				Level:    slog.LevelInfo,
				Format:   "json",
				Output:   "file",
				FilePath: "/tmp/test.log",
			},
			wantErr: false,
		},
		{
			name: "valid sampling config",
			cfg: Config{
				Level:  slog.LevelInfo,
				Format: "json",
				Output: "stdout",
				Sampling: SamplingConfig{
					Enabled:   true,
					Type:      "level",
					InfoRate:  0.5,
					DebugRate: 0.1,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid sampling rates",
			cfg: Config{
				Level:  slog.LevelInfo,
				Format: "json",
				Output: "stdout",
				Sampling: SamplingConfig{
					Enabled:   true,
					Type:      "level",
					InfoRate:  1.5, // > 1.0
					DebugRate: 0.1,
				},
			},
			wantErr: true,
		},
		{
			name: "negative sampling rate",
			cfg: Config{
				Level:  slog.LevelInfo,
				Format: "json",
				Output: "stdout",
				Sampling: SamplingConfig{
					Enabled:   true,
					Type:      "level",
					InfoRate:  -0.5,
					DebugRate: 0.1,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid debug sampling rate",
			cfg: Config{
				Level:  slog.LevelInfo,
				Format: "json",
				Output: "stdout",
				Sampling: SamplingConfig{
					Enabled:   true,
					Type:      "level",
					InfoRate:  0.5,
					DebugRate: 2.0, // > 1.0
				},
			},
			wantErr: true,
		},
		{
			name: "rate sampling with invalid rate",
			cfg: Config{
				Level:  slog.LevelInfo,
				Format: "json",
				Output: "stdout",
				Sampling: SamplingConfig{
					Enabled: true,
					Type:    "rate",
					Rate:    1.5, // > 1.0
				},
			},
			wantErr: true,
		},
		{
			name: "rate sampling with negative rate",
			cfg: Config{
				Level:  slog.LevelInfo,
				Format: "json",
				Output: "stdout",
				Sampling: SamplingConfig{
					Enabled: true,
					Type:    "rate",
					Rate:    -0.5, // < 0
				},
			},
			wantErr: true,
		},
		{
			name: "adaptive sampling with invalid target rate",
			cfg: Config{
				Level:  slog.LevelInfo,
				Format: "json",
				Output: "stdout",
				Sampling: SamplingConfig{
					Enabled:    true,
					Type:       "adaptive",
					TargetRate: 0, // <= 0
				},
			},
			wantErr: true,
		},
		{
			name: "adaptive sampling with valid config",
			cfg: Config{
				Level:  slog.LevelInfo,
				Format: "json",
				Output: "stdout",
				Sampling: SamplingConfig{
					Enabled:    true,
					Type:       "adaptive",
					TargetRate: 1000,
					Window:     time.Minute,
				},
			},
			wantErr: false,
		},
		{
			name: "adaptive sampling with zero window gets default",
			cfg: Config{
				Level:  slog.LevelInfo,
				Format: "json",
				Output: "stdout",
				Sampling: SamplingConfig{
					Enabled:    true,
					Type:       "adaptive",
					TargetRate: 1000,
					Window:     0, // Should get default
				},
			},
			wantErr: false,
		},
		{
			name: "invalid sampling type",
			cfg: Config{
				Level:  slog.LevelInfo,
				Format: "json",
				Output: "stdout",
				Sampling: SamplingConfig{
					Enabled: true,
					Type:    "invalid",
				},
			},
			wantErr: true,
		},
		{
			name: "multi output with file path",
			cfg: Config{
				Level:    slog.LevelInfo,
				Format:   "json",
				Output:   "multi",
				FilePath: "/tmp/test.log",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewConfigWithOptions(t *testing.T) {
	t.Run("WithLevel", func(t *testing.T) {
		cfg := NewConfig(WithLevel(slog.LevelDebug))
		if cfg.Level != slog.LevelDebug {
			t.Errorf("Expected debug level, got %v", cfg.Level)
		}
	})

	t.Run("WithFormat", func(t *testing.T) {
		cfg := NewConfig(WithFormat("pretty"))
		if cfg.Format != "pretty" {
			t.Errorf("Expected pretty format, got %s", cfg.Format)
		}
	})

	t.Run("WithOutput", func(t *testing.T) {
		cfg := NewConfig(WithOutput("file"))
		if cfg.Output != "file" {
			t.Errorf("Expected file output, got %s", cfg.Output)
		}
	})

	t.Run("WithFile", func(t *testing.T) {
		cfg := NewConfig(WithFile("/tmp/test.log", 1024*1024, 7, 3))
		if cfg.FilePath != "/tmp/test.log" {
			t.Errorf("Expected /tmp/test.log, got %s", cfg.FilePath)
		}
		if cfg.MaxSize != 1024*1024 {
			t.Errorf("Expected max size 1MB, got %d", cfg.MaxSize)
		}
		if cfg.MaxAge != 7 {
			t.Errorf("Expected max age 7, got %d", cfg.MaxAge)
		}
		if cfg.MaxBackups != 3 {
			t.Errorf("Expected max backups 3, got %d", cfg.MaxBackups)
		}
	})

	t.Run("WithSampling", func(t *testing.T) {
		cfg := NewConfig(WithSampling(true, "level"))
		if !cfg.Sampling.Enabled {
			t.Error("Expected sampling enabled")
		}
		if cfg.Sampling.Type != "level" {
			t.Errorf("Expected level sampling type, got %s", cfg.Sampling.Type)
		}
	})

	t.Run("WithDevelopment", func(t *testing.T) {
		cfg := NewConfig(WithDevelopment(true))
		if !cfg.Development {
			t.Error("Expected development mode")
		}
		if !cfg.AddSource {
			t.Error("Expected AddSource enabled")
		}
		if cfg.EnableSensitive {
			t.Error("Expected EnableSensitive disabled")
		}
	})
}
