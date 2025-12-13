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
	NATS           NATSConfig           `mapstructure:"nats"`
}

type ServerConfig struct {
	Port string `mapstructure:"port"`
}

type ProjectServiceConfig struct {
	GrpcAddress string `mapstructure:"grpc"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"name"`
}

type NATSConfig struct {
	URL     string `mapstructure:"url"`
	Subject string `mapstructure:"subject"`
}

func Load() (*Config, error) {
	// Get environment from ENV, default to "local"
	env := os.Getenv("ENV")
	if env == "" {
		env = "local"
	}

	// Set up Viper
	viper.SetConfigName(fmt.Sprintf("config.%s", env))
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/configs")                           // Kubernetes mount
	viper.AddConfigPath("./services/student-service/configs") // IDE from root
	viper.AddConfigPath("../configs")                         // IDE from cmd/server

	// Try to read config file (optional - will use ENV if not found)
	if err := viper.ReadInConfig(); err != nil {
		// Config file is optional - continue with ENV variables
		fmt.Printf("No config file found (will use ENV variables): %v\n", err)
	}

	// Enable environment variable overrides (these take precedence over config file)
	viper.AutomaticEnv()

	viper.BindEnv("database.user", "DB_USER")
	viper.BindEnv("database.password", "DB_PASSWORD")

	// Unmarshal into struct
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}
