package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lizhiqiang-1996/go_doctor/pkg/types"
)

func TestLoadConfig(t *testing.T) {
	dir := os.TempDir()
	configPath := filepath.Join(dir, "go-doctor.config.json")
	configContent := `{
		"ignore": {
			"rules": ["style/exported-without-comment"],
			"files": ["generated/**"]
		}
	}`
	os.WriteFile(configPath, []byte(configContent), 0644)
	defer os.Remove(configPath)

	cfg := Load(dir)
	if cfg == nil {
		t.Fatal("Config should not be nil")
	}
	if cfg.Ignore == nil {
		t.Fatal("Ignore config should not be nil")
	}
	if len(cfg.Ignore.Rules) != 1 {
		t.Errorf("Expected 1 ignore rule, got %d", len(cfg.Ignore.Rules))
	}
	if cfg.Ignore.Rules[0] != "style/exported-without-comment" {
		t.Errorf("Unexpected rule: %s", cfg.Ignore.Rules[0])
	}
}

func TestLoadMissingConfig(t *testing.T) {
	dir := os.TempDir()
	cfg := Load(filepath.Join(dir, "nonexistent"))
	if cfg == nil {
		t.Fatal("Config should return empty config for missing file")
	}
}

func TestMergeWithDefaults(t *testing.T) {
	cfg := &types.Config{}
	merged := MergeWithDefaults(cfg)

	if merged.Lint == nil || !*merged.Lint {
		t.Error("Default lint should be true")
	}
	if merged.DeadCode == nil || !*merged.DeadCode {
		t.Error("Default deadCode should be true")
	}
	if merged.Verbose == nil || *merged.Verbose {
		t.Error("Default verbose should be false")
	}
}

func TestGetIgnoreRules(t *testing.T) {
	cfg := &types.Config{
		Ignore: &types.IgnoreConfig{
			Rules: []string{"rule1", "rule2"},
		},
	}
	rules := GetIgnoreRules(cfg)
	if len(rules) != 2 {
		t.Errorf("Expected 2 rules, got %d", len(rules))
	}
}

func TestGetIgnoreRulesNil(t *testing.T) {
	rules := GetIgnoreRules(nil)
	if rules != nil {
		t.Error("Expected nil for nil config")
	}
}
