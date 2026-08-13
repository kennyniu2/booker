package config

import "testing"

func TestLoadWithoutEnvFile(t *testing.T) {
	t.Chdir(t.TempDir()) // no .env here
	t.Setenv("DB_HOST", "db")
	t.Setenv("DISCORD_TOKEN", "tok")
	t.Setenv("API_ENABLED", "false")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.DBHost != "db" || cfg.DiscordToken != "tok" {
		t.Errorf("env vars not read: %+v", cfg)
	}
	if cfg.APIEnabled {
		t.Error("APIEnabled should be false")
	}
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want default 8080", cfg.Port)
	}
}

func TestAPIEnabledByDefault(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("API_ENABLED", "")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.APIEnabled {
		t.Error("APIEnabled should default to true")
	}
}
