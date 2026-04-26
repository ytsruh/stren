package utils

import (
	"testing"
)

func TestLoadAndValidateEnv_Success(t *testing.T) {
	// Ensure both required variables are set
	t.Setenv("PORT", "8080")
	t.Setenv("DB_PATH", "test.db")

	cfg, err := LoadAndValidateEnv()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if cfg.PORT != "8080" {
		t.Errorf("expected PORT to be '8080', got: %s", cfg.PORT)
	}
	if cfg.DB_PATH != "test.db" {
		t.Errorf("expected DB_PATH to be 'test.db', got: %s", cfg.DB_PATH)
	}
}

func TestLoadAndValidateEnv_MissingPort(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("DB_PATH", "test.db")

	_, err := LoadAndValidateEnv()
	if err == nil {
		t.Fatal("expected error for missing PORT, got nil")
	}
}

func TestLoadAndValidateEnv_MissingDBPath(t *testing.T) {
	t.Setenv("PORT", "8080")
	t.Setenv("DB_PATH", "")

	_, err := LoadAndValidateEnv()
	if err == nil {
		t.Fatal("expected error for missing DB_PATH, got nil")
	}
}

func TestLoadAndValidateEnv_MissingBoth(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("DB_PATH", "")

	_, err := LoadAndValidateEnv()
	if err == nil {
		t.Fatal("expected error for missing variables, got nil")
	}
}

func TestValidateEnvVars(t *testing.T) {
	tests := []struct {
		name     string
		env      EnvVar
		expected []string
	}{
		{
			name:     "all fields present",
			env:      EnvVar{PORT: "8080", DB_PATH: "test.db"},
			expected: []string{},
		},
		{
			name:     "missing port",
			env:      EnvVar{PORT: "", DB_PATH: "test.db"},
			expected: []string{"PORT"},
		},
		{
			name:     "missing db_path",
			env:      EnvVar{PORT: "8080", DB_PATH: ""},
			expected: []string{"DB_PATH"},
		},
		{
			name:     "missing both",
			env:      EnvVar{PORT: "", DB_PATH: ""},
			expected: []string{"PORT", "DB_PATH"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			missing := ValidateEnvVars(tt.env)
			if len(missing) != len(tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, missing)
			}
			for i, v := range tt.expected {
				if missing[i] != v {
					t.Errorf("expected missing[%d] to be %s, got %s", i, v, missing[i])
				}
			}
		})
	}
}

func TestGetEnvVars(t *testing.T) {
	// Set up a valid config first
	t.Setenv("PORT", "9090")
	t.Setenv("DB_PATH", "getenv.db")

	_, err := LoadAndValidateEnv()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	cfg := GetEnvVars()
	if cfg == nil {
		t.Fatal("expected config, got nil")
	}
	if cfg.PORT != "9090" {
		t.Errorf("expected PORT '9090', got '%s'", cfg.PORT)
	}
	if cfg.DB_PATH != "getenv.db" {
		t.Errorf("expected DB_PATH 'getenv.db', got '%s'", cfg.DB_PATH)
	}
}

func TestGetEnvVars_NotLoaded(t *testing.T) {
	// Reset global state by setting config to nil
	configMu.Lock()
	config = nil
	configMu.Unlock()

	cfg := GetEnvVars()
	if cfg != nil {
		t.Error("expected nil when config not loaded, got a value")
	}
}
