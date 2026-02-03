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
	AutoVerticalOutput bool              `mapstructure:"auto_vertical_output"`
	MultiLine          bool              `mapstructure:"multi_line"`
	MaxResults         int               `mapstructure:"max_results"`
}

func LoadConfig(customPath string) (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not find home directory: %v", err)
	}

	configDir := filepath.Join(home, ".config", "mytui")

	viper.SetDefault("table_format", "unicode")
	viper.SetDefault("syntax_style", "monokai")
	viper.SetDefault("key_bindings", "vim")
	viper.SetDefault("history_file", filepath.Join(configDir, "history"))
	viper.SetDefault("smart_completion", true)
	viper.SetDefault("timing", true)
	viper.SetDefault("pager", "less -SRXF")
	viper.SetDefault("prompt", "\\t \\u@\\x1b[34m\\h\\x1b[0m:\\d> ")
	viper.SetDefault("prompt_continuation", "-> ")
	viper.SetDefault("favorite_queries", make(map[string]string))
	viper.SetDefault("auto_vertical_output", true)
	viper.SetDefault("multi_line", false)
	viper.SetDefault("max_results", 5)

	if customPath != "" {
		viper.SetConfigFile(customPath)
	} else {
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return nil, fmt.Errorf("could not create config directory: %v", err)
		}
		viper.SetConfigName("config")
		viper.SetConfigType("toml")
		viper.AddConfigPath(configDir)
		viper.AddConfigPath(home)
	}

	viper.SetEnvPrefix("SQLCLI")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok && customPath != "" {
			return nil, fmt.Errorf("error reading config file: %v", err)
		}
		// If not custom path and not found, we can try to create default
		if customPath == "" {
			if _, err := os.Stat(filepath.Join(configDir, "config.toml")); os.IsNotExist(err) {
				_ = viper.SafeWriteConfigAs(filepath.Join(configDir, "config.toml"))
			}
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unable to decode into struct: %v", err)
	}

	return &cfg, nil
}
