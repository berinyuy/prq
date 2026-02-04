package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	Provider  ProviderConfig  `mapstructure:"provider"`
	UserRules []string        `mapstructure:"user_rules"`
	Queue     QueueConfig     `mapstructure:"queue"`
	Redaction RedactionConfig `mapstructure:"redaction"`
	TUI       TUIConfig       `mapstructure:"tui"`
}

type ProviderConfig struct {
	Command string   `mapstructure:"command"`
	Args    []string `mapstructure:"args"`
}

type QueueConfig struct {
	DefaultLimit int    `mapstructure:"default_limit"`
	DefaultSort  string `mapstructure:"default_sort"`
}

type RedactionConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

type TUIConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

type RepoConfig struct {
	RepoRules []string    `mapstructure:"repo_rules"`
	Tests     TestsConfig `mapstructure:"tests"`
	Diff      DiffConfig  `mapstructure:"diff"`
}

type TestsConfig struct {
	Commands []string `mapstructure:"commands"`
}

type DiffConfig struct {
	Ignore        []string `mapstructure:"ignore"`
	MaxFiles      int      `mapstructure:"max_files"`
	MaxChunkChars int      `mapstructure:"max_chunk_chars"`
}

func Defaults() Config {
	return Config{
		Provider: ProviderConfig{
			Command: "claude",
			Args:    []string{},
		},
		Queue: QueueConfig{
			DefaultLimit: 200,
			DefaultSort:  "oldest",
		},
		Redaction: RedactionConfig{Enabled: true},
		TUI:       TUIConfig{Enabled: true},
	}
}

func DefaultRepoConfig() RepoConfig {
	return RepoConfig{
		RepoRules: []string{},
		Tests:     TestsConfig{Commands: []string{}},
		Diff: DiffConfig{
			Ignore:        []string{},
			MaxFiles:      50,
			MaxChunkChars: 8000,
		},
	}
}

func Load(configPath string) (Config, RepoConfig, error) {
	userCfg := Defaults()
	repoCfg := DefaultRepoConfig()

	if err := loadUserConfig(configPath, &userCfg); err != nil {
		return Config{}, RepoConfig{}, err
	}
	if err := loadRepoConfig(&repoCfg); err != nil {
		return Config{}, RepoConfig{}, err
	}

	if userCfg.Provider.Command == "" {
		userCfg.Provider.Command = "claude"
	}
	if userCfg.Queue.DefaultLimit == 0 {
		userCfg.Queue.DefaultLimit = 200
	}
	if userCfg.Queue.DefaultSort == "" {
		userCfg.Queue.DefaultSort = "oldest"
	}
	if repoCfg.Diff.MaxFiles == 0 {
		repoCfg.Diff.MaxFiles = 50
	}
	if repoCfg.Diff.MaxChunkChars == 0 {
		repoCfg.Diff.MaxChunkChars = 8000
	}

	return userCfg, repoCfg, nil
}

func loadUserConfig(configPath string, cfg *Config) error {
	path := configPath
	if path == "" {
		path = filepath.Join(os.Getenv("HOME"), ".prq", "config.yaml")
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read user config: %w", err)
	}

	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to load user config: %w", err)
	}
	if err := v.Unmarshal(cfg); err != nil {
		return fmt.Errorf("failed to parse user config: %w", err)
	}
	return nil
}

func loadRepoConfig(cfg *RepoConfig) error {
	path := filepath.Join(".", "prq.yaml")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read repo config: %w", err)
	}

	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to load repo config: %w", err)
	}
	if err := v.Unmarshal(cfg); err != nil {
		return fmt.Errorf("failed to parse repo config: %w", err)
	}
	return nil
}
