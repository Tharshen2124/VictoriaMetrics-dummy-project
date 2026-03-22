package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	Port        string `mapstructure:"PORT"`
	DatabaseURL string `mapstructure:"DATABASE_URL"`
}

func LoadConfig() (*Config, error) {
	v := viper.New()
	v.SetConfigFile(".env")
	v.SetConfigType("env")

	if err := v.BindEnv("PORT"); err != nil {
		return nil, fmt.Errorf("bind PORT: %w", err)
	}

	if err := v.BindEnv("DATABASE_URL"); err != nil {
		return nil, fmt.Errorf("bind DATABASE_URL: %w", err)
	}

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read .env: %w", err)
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return &config, nil
}
