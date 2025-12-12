package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	Env      string         `mapstructure:"env"`
	Database DatabaseConfig `mapstructure:"database"`
	Grpc     GrpcConfig     `mapstructure:"grpc"`
	Kafka    KafkaConfig    `mapstructure:"kafka"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
}

type GrpcConfig struct {
	Port string `mapstructure:"port"`
}

type KafkaConfig struct {
	Brokers []string `mapstructure:"brokers"`
	Topic   string   `mapstructure:"topic"`
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
	viper.AddConfigPath("/configs")                           // Kubernetes mount
	viper.AddConfigPath("./services/project-service/configs") // IDE from root
	viper.AddConfigPath("../configs")                         // IDE from cmd/server

	// Try to read config file (optional - will use ENV if not found)
	if err := viper.ReadInConfig(); err != nil {
		// Config file is optional - continue with ENV variables
		fmt.Printf("No config file found (will use ENV variables): %v\n", err)
	}

	// Enable environment variable overrides (these take precedence over config file)
	viper.AutomaticEnv()

	// Bind ONLY sensitive data from environment variables (Secrets)
	// Other config comes from the config file (ConfigMap)
	viper.BindEnv("database.user", "DB_USER")
	viper.BindEnv("database.password", "DB_PASSWORD")

	// Unmarshal into struct
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}
