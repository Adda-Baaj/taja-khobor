package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config holds application configuration pulled from environment variables / viper.
type Config struct {
	AppName        string        `mapstructure:"app_name"`
	Env            string        `mapstructure:"app_env"`
	LogLevel       string        `mapstructure:"log_level"`
	ProvidersFile  string        `mapstructure:"providers_file"`
	PublishersFile string        `mapstructure:"publishers_file"`
	CrawlInterval  time.Duration `mapstructure:"crawl_interval"`

	StorageType string `mapstructure:"storage_type"`
	BBoltPath   string `mapstructure:"bbolt_path"`
}

// Load reads configuration. It will load optional configs/.env (for local dev), then use viper to read
// environment variables and defaults. Returns typed Config or error.
func Load() (*Config, error) {
	// load optional .env file
	_ = godotenv.Load("configs/.env")

	v := viper.New()

	// defaults
	v.SetDefault("app_name", "taja-khobor")
	v.SetDefault("app_env", "development")
	v.SetDefault("log_level", "info")
	v.SetDefault("providers_file", "./configs/providers.yaml")
	v.SetDefault("publishers_file", "./configs/publishers.yaml")
	v.SetDefault("crawl_interval", "15m")
	v.SetDefault("storage_type", "bbolt")
	v.SetDefault("bbolt_path", "./data/cache.db")

	// allow env variables to override; use prefix TAJA (optional)
	v.AutomaticEnv()

	// Unmarshal into struct; viper will read from environment variables too.
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	// post-process crawl_interval: accept either duration string or integer minutes
	if d := v.GetDuration("crawl_interval"); d > 0 {
		cfg.CrawlInterval = d
	} else if s := strings.TrimSpace(v.GetString("crawl_interval")); s != "" {
		if parsed, err := time.ParseDuration(s); err == nil {
			cfg.CrawlInterval = parsed
		} else {
			var minutes int
			if _, scanErr := fmt.Sscanf(s, "%d", &minutes); scanErr != nil {
				return nil, fmt.Errorf("invalid crawl_interval: %s", s)
			}
			cfg.CrawlInterval = time.Duration(minutes) * time.Minute
		}
	}

	return &cfg, nil
}
