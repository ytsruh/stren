package utils

import (
	"testing"
)

func TestLoadAndValidateEnv_Success(t *testing.T) {
	// Ensure all required variables are set
	t.Setenv("PORT", "8080")
	t.Setenv("DB_PATH", "test.db")
	t.Setenv("TURSO_DATABASE_URL", "libsql://test.turso.io")
	t.Setenv("TURSO_AUTH_TOKEN", "test-token")
	t.Setenv("JWT_SECRET", "test-secret")

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
	if cfg.TURSO_DATABASE_URL != "libsql://test.turso.io" {
		t.Errorf("expected TURSO_DATABASE_URL to be 'libsql://test.turso.io', got: %s", cfg.TURSO_DATABASE_URL)
	}
	if cfg.TURSO_AUTH_TOKEN != "test-token" {
		t.Errorf("expected TURSO_AUTH_TOKEN to be 'test-token', got: %s", cfg.TURSO_AUTH_TOKEN)
	}
	if cfg.JWT_SECRET != "test-secret" {
		t.Errorf("expected JWT_SECRET to be 'test-secret', got: %s", cfg.JWT_SECRET)
	}
}

func TestLoadAndValidateEnv_MissingPort(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("DB_PATH", "test.db")
	t.Setenv("TURSO_DATABASE_URL", "libsql://test.turso.io")
	t.Setenv("TURSO_AUTH_TOKEN", "test-token")
	t.Setenv("JWT_SECRET", "test-secret")

	_, err := LoadAndValidateEnv()
	if err == nil {
		t.Fatal("expected error for missing PORT, got nil")
	}
}

func TestLoadAndValidateEnv_MissingDBPath(t *testing.T) {
	t.Setenv("PORT", "8080")
	t.Setenv("DB_PATH", "")
	t.Setenv("TURSO_DATABASE_URL", "libsql://test.turso.io")
	t.Setenv("TURSO_AUTH_TOKEN", "test-token")
	t.Setenv("JWT_SECRET", "test-secret")

	_, err := LoadAndValidateEnv()
	if err == nil {
		t.Fatal("expected error for missing DB_PATH, got nil")
	}
}

func TestLoadAndValidateEnv_MissingBoth(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("DB_PATH", "")
	t.Setenv("TURSO_DATABASE_URL", "")
	t.Setenv("TURSO_AUTH_TOKEN", "")
	t.Setenv("JWT_SECRET", "")

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
			env:      EnvVar{PORT: "8080", DB_PATH: "test.db", TURSO_DATABASE_URL: "libsql://test.turso.io", TURSO_AUTH_TOKEN: "test-token", JWT_SECRET: "test-secret"},
			expected: []string{},
		},
		{
			name:     "missing port",
			env:      EnvVar{PORT: "", DB_PATH: "test.db", TURSO_DATABASE_URL: "libsql://test.turso.io", TURSO_AUTH_TOKEN: "test-token", JWT_SECRET: "test-secret"},
			expected: []string{"PORT"},
		},
		{
			name:     "missing db_path",
			env:      EnvVar{PORT: "8080", DB_PATH: "", TURSO_DATABASE_URL: "libsql://test.turso.io", TURSO_AUTH_TOKEN: "test-token", JWT_SECRET: "test-secret"},
			expected: []string{"DB_PATH"},
		},
		{
			name:     "missing both",
			env:      EnvVar{PORT: "", DB_PATH: "", TURSO_DATABASE_URL: "libsql://test.turso.io", TURSO_AUTH_TOKEN: "test-token", JWT_SECRET: "test-secret"},
			expected: []string{"PORT", "DB_PATH"},
		},
		{
			name:     "missing turso url",
			env:      EnvVar{PORT: "8080", DB_PATH: "test.db", TURSO_DATABASE_URL: "", TURSO_AUTH_TOKEN: "test-token", JWT_SECRET: "test-secret"},
			expected: []string{"TURSO_DATABASE_URL"},
		},
		{
			name:     "missing turso token",
			env:      EnvVar{PORT: "8080", DB_PATH: "test.db", TURSO_DATABASE_URL: "libsql://test.turso.io", TURSO_AUTH_TOKEN: "", JWT_SECRET: "test-secret"},
			expected: []string{"TURSO_AUTH_TOKEN"},
		},
		{
			name:     "missing jwt secret",
			env:      EnvVar{PORT: "8080", DB_PATH: "test.db", TURSO_DATABASE_URL: "libsql://test.turso.io", TURSO_AUTH_TOKEN: "test-token", JWT_SECRET: ""},
			expected: []string{"JWT_SECRET"},
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
	t.Setenv("TURSO_DATABASE_URL", "libsql://getenv.turso.io")
	t.Setenv("TURSO_AUTH_TOKEN", "getenv-token")
	t.Setenv("JWT_SECRET", "getenv-secret")

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
	if cfg.TURSO_DATABASE_URL != "libsql://getenv.turso.io" {
		t.Errorf("expected TURSO_DATABASE_URL 'libsql://getenv.turso.io', got '%s'", cfg.TURSO_DATABASE_URL)
	}
	if cfg.TURSO_AUTH_TOKEN != "getenv-token" {
		t.Errorf("expected TURSO_AUTH_TOKEN 'getenv-token', got '%s'", cfg.TURSO_AUTH_TOKEN)
	}
	if cfg.JWT_SECRET != "getenv-secret" {
		t.Errorf("expected JWT_SECRET 'getenv-secret', got '%s'", cfg.JWT_SECRET)
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
