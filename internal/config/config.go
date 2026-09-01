package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ConfigDir           string      `yaml:"configDir,omitempty"`
	DebounceInMs        int         `yaml:"debounceInMs,omitempty"`
	PreservePermissions bool        `yaml:"preservePermissions,omitempty"`
	Checksum            bool        `yaml:"checksum,omitempty"`
	Workers             int         `yaml:"workers,omitempty"`
	Ignore              []string    `yaml:"ignore,omitempty"`
	Sync                []Directory `yaml:"sync"`
}

type Directory struct {
	Title          string   `yaml:"title"`
	Source         string   `yaml:"source"`
	Destination    string   `yaml:"destination"`
	Ignore         []string `yaml:"ignore,omitempty"`
	Once           bool     `yaml:"once,omitempty"`
	OnSourceDelete string   `yaml:"onSourceDelete,omitempty"` // restore (default) or remove
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found: %s", path)
		}
		return nil, fmt.Errorf("cannot read config %q: %w", path, err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("invalid YAML in %q: %w", path, err)
	}

	if len(config.Sync) == 0 {
		return nil, fmt.Errorf("config has no 'sync' entries")
	}

	for i, d := range config.Sync {
		if d.Source == "" || d.Destination == "" {
			name := d.Title
			if name == "" {
				name = fmt.Sprintf("#%d", i+1)
			}
			return nil, fmt.Errorf("sync entry %s is missing a source or destination", name)
		}
	}

	for i := range config.Sync {
		d := &config.Sync[i]
		if d.OnSourceDelete == "" {
			d.OnSourceDelete = "restore"
		}
		if d.OnSourceDelete != "restore" && d.OnSourceDelete != "remove" {
			name := d.Title
			if name == "" {
				name = fmt.Sprintf("#%d", i+1)
			}
			return nil, fmt.Errorf("sync entry %s onSourceDelete must be restore or remove", name)
		}
	}

	if config.ConfigDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("cannot determine working directory: %w", err)
		}
		config.ConfigDir = filepath.Join(cwd, ".gofilesync")
	}

	if config.DebounceInMs == 0 {
		config.DebounceInMs = 300
	}

	if config.Workers <= 0 {
		config.Workers = 8
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

func (c *Config) GetManifestFile() string {
	return filepath.Join(c.ConfigDir, "managed.json")
}
