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
	viper.AddConfigPath("./configs")                          // Docker runtime
	viper.AddConfigPath("./services/project-service/configs") // IDE from root
	viper.AddConfigPath("./project-service/configs")          // Legacy path
	viper.AddConfigPath("../configs")                         // IDE from cmd/server
	viper.AddConfigPath("../../configs")                      // IDE from other locations

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Enable environment variable overrides
	viper.AutomaticEnv()
	viper.BindEnv("grpc.port", "GRPC_PORT")

	// Unmarshal into struct
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}
