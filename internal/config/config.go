package config

import (
	"fmt"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	AppName              string        `mapstructure:"app_name"`
	Env                  string        `mapstructure:"app_env"`
	LogLevel             string        `mapstructure:"log_level"`
	ProvidersFile        string        `mapstructure:"providers_file"`
	PublishersFile       string        `mapstructure:"publishers_file"`
	CrawlIntervalSeconds int64         `mapstructure:"crawl_interval"`
	CrawlInterval        time.Duration `mapstructure:"-"`

	StorageType string `mapstructure:"storage_type"`
	BBoltPath   string `mapstructure:"bbolt_path"`
}

func Load() (*Config, error) {
	_ = godotenv.Load("configs/.env")

	v := viper.New()

	v.SetDefault("app_name", "taja-khobor")
	v.SetDefault("app_env", "development")
	v.SetDefault("log_level", "info")
	v.SetDefault("providers_file", "./configs/providers.yaml")
	v.SetDefault("publishers_file", "./configs/publishers.yaml")
	v.SetDefault("crawl_interval", 900) // seconds
	v.SetDefault("storage_type", "bbolt")
	v.SetDefault("bbolt_path", "./data/cache.db")

	v.AutomaticEnv()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	if cfg.CrawlIntervalSeconds <= 0 {
		return nil, fmt.Errorf("invalid crawl_interval (must be positive seconds)")
	}
	cfg.CrawlInterval = time.Duration(cfg.CrawlIntervalSeconds) * time.Second

	return &cfg, nil
}
