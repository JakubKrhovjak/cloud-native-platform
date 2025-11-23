package config

import "os"

type Config struct {
	Server         ServerConfig
	Database       DatabaseConfig
	ProjectService ProjectServiceConfig
}

type ServerConfig struct {
	Port string
}

type ProjectServiceConfig struct {
	BaseURL     string
	GrpcAddress string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8080"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5439"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			DBName:   getEnv("DB_NAME", "university"),
		},
		ProjectService: ProjectServiceConfig{
			BaseURL:     getEnv("PROJECT_SERVICE_URL", "http://localhost:8081"),
			GrpcAddress: getEnv("PROJECT_SERVICE_GRPC", "localhost:9090"),
		},
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
