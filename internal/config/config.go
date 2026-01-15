package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	TableFormat        string            `mapstructure:"table_format"`
	SyntaxStyle        string            `mapstructure:"syntax_style"`
	KeyBindings        string            `mapstructure:"key_bindings"`
	HistoryFile        string            `mapstructure:"history_file"`
	SmartCompletion    bool              `mapstructure:"smart_completion"`
	Timing             bool              `mapstructure:"timing"`
	Pager              string            `mapstructure:"pager"`
	Prompt             string            `mapstructure:"prompt"`
	PromptContinuation string            `mapstructure:"prompt_continuation"`
	FavoriteQueries    map[string]string `mapstructure:"favorite_queries"`
}

func LoadConfig() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not find home directory: %v", err)
	}

	configDir := filepath.Join(home, ".config", "sqlcli")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("could not create config directory: %v", err)
	}

	viper.SetDefault("table_format", "table")
	viper.SetDefault("syntax_style", "monokai")
	viper.SetDefault("key_bindings", "vim")
	viper.SetDefault("history_file", filepath.Join(configDir, "history"))
	viper.SetDefault("smart_completion", true)
	viper.SetDefault("timing", true)
	viper.SetDefault("pager", "less -SRXF")
	viper.SetDefault("prompt", "\\t \\u@\\x1b[34m\\h\\x1b[0m:\\d> ")
	viper.SetDefault("prompt_continuation", "-> ")
	viper.SetDefault("favorite_queries", make(map[string]string))

	viper.SetConfigName("config")
	viper.SetConfigType("toml") // We'll use TOML as default for now, but viper is flexible
	viper.AddConfigPath(configDir)
	viper.AddConfigPath(home)
	viper.SetEnvPrefix("SQLCLI")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %v", err)
		}
		// Config file not found, use defaults and optionally create one
		if err := viper.SafeWriteConfigAs(filepath.Join(configDir, "config.toml")); err != nil {
			// Ignore error if it already exists, although SafeWriteConfigAs shouldn't be called if it exists
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unable to decode into struct: %v", err)
	}

	return &cfg, nil
}
