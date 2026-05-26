package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/go-doctor/go-doctor/pkg/types"
)

const ConfigFileName = "go-doctor.config.json"

func Load(rootDir string) *types.Config {
	configPath := filepath.Join(rootDir, ConfigFileName)
	data, err := os.ReadFile(configPath)
	if err != nil {
		return &types.Config{}
	}

	var cfg types.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return &types.Config{}
	}

	return &cfg
}

func MergeWithDefaults(cfg *types.Config) *types.Config {
	if cfg == nil {
		cfg = &types.Config{}
	}

	result := *cfg

	if result.Lint == nil {
		lint := true
		result.Lint = &lint
	}

	if result.DeadCode == nil {
		deadCode := true
		result.DeadCode = &deadCode
	}

	if result.Verbose == nil {
		verbose := false
		result.Verbose = &verbose
	}

	if result.Ignore == nil {
		result.Ignore = &types.IgnoreConfig{
			Rules: []string{},
			Files: []string{},
		}
	}

	return &result
}

func GetIgnoreRules(cfg *types.Config) []string {
	if cfg == nil || cfg.Ignore == nil {
		return nil
	}
	return cfg.Ignore.Rules
}

func GetIgnoreFiles(cfg *types.Config) []string {
	if cfg == nil || cfg.Ignore == nil {
		return nil
	}
	return cfg.Ignore.Files
}
