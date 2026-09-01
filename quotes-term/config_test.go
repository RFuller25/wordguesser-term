package main

import "testing"

func TestConfigRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cfg := &Config{APIKey: "secret123", Username: "rhys"}
	if err := saveConfig(cfg); err != nil {
		t.Fatalf("saveConfig failed: %v", err)
	}

	loaded, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig failed: %v", err)
	}
	if loaded.APIKey != cfg.APIKey || loaded.Username != cfg.Username {
		t.Errorf("round trip mismatch: got %+v, want %+v", loaded, cfg)
	}
}

func TestLoadConfigMissing(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	_, err := loadConfig()
	if err == nil {
		t.Fatal("expected error loading missing config, got nil")
	}
}
