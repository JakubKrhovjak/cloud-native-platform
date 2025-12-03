package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	Env            string               `mapstructure:"env"`
	Server         ServerConfig         `mapstructure:"server"`
	Database       DatabaseConfig       `mapstructure:"database"`
	ProjectService ProjectServiceConfig `mapstructure:"project_service"`
}

type ServerConfig struct {
	Port string `mapstructure:"port"`
}

type ProjectServiceConfig struct {
	BaseURL     string `mapstructure:"url"`
	GrpcAddress string `mapstructure:"grpc"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"name"`
}

func Load() (*Config, error) {
	// Get environment from APP_ENV, default to "local"
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "local"
	}

	// Set up Viper
	viper.SetConfigName(fmt.Sprintf("config.%s", env))
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")                          // Docker runtime
	viper.AddConfigPath("./services/student-service/configs") // IDE from root
	viper.AddConfigPath("../configs")                         // IDE from cmd/server
	viper.AddConfigPath("../../configs")                      // IDE from other locations

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Unmarshal into struct
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}
