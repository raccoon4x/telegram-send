package config

import (
	"fmt"
	"github.com/spf13/viper"
)

type Config struct {
	Telegram struct {
		Token  string
		ChatID string
	}
}

func LoadConfig(configPaths ...string) (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	for _, path := range configPaths {
		viper.AddConfigPath(path)
	}
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unable to decode into struct: %w", err)
	}

	return &cfg, nil
}
