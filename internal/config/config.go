package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ConfigDir    string      `yaml:"configDir,omitempty"`
	DebounceInMs int         `yaml:"debounceInMs,omitempty"`
	Sync         []Directory `yaml:"sync"`
}

type Directory struct {
	Title       string `yaml:"title"`
	Source      string `yaml:"source"`
	Destination string `yaml:"destination"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	if len(config.Sync) == 0 {
		return nil, fmt.Errorf("no sync entries defined in configuration")
	}

	if config.ConfigDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %v", err)
		}
		config.ConfigDir = filepath.Join(cwd, ".gofilesync")
	}

	if config.DebounceInMs == 0 {
		config.DebounceInMs = 300
	}

	return &config, nil
}

func (c *Config) GetStateDir() string {
	return c.ConfigDir
}

func (c *Config) GetBackupDir() string {
	return filepath.Join(c.ConfigDir, ".backups")
}

func (c *Config) GetLogFile() string {
	return filepath.Join(c.ConfigDir, "sync.log")
}
