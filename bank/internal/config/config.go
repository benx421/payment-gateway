package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration
type Config struct {
	Server   ServerConfig
	Logger   LoggerConfig
	Database DatabaseConfig
	App      AppConfig
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DatabaseConfig holds database connection configuration
type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	DBName          string
	SSLMode         string
	ConnMaxLifetime time.Duration
	MaxOpenConns    int
	MaxIdleConns    int
}

// AppConfig holds application-specific configuration
type AppConfig struct {
	FailureRate        float64
	MinLatencyMS       int
	MaxLatencyMS       int
	AuthExpiryHours    int
	AuthExpiryDuration time.Duration
}

// LoggerConfig holds logging configuration
type LoggerConfig struct {
	Level string // debug, info, warn, error
}

// Load loads configuration from environment variables with sensible defaults
func Load() (*Config, error) {
	authExpiryHours := getEnvAsInt("AUTH_EXPIRY_HOURS", 168) // 7 days default

	cfg := &Config{
		Server: ServerConfig{
			Port:         getEnv("PORT", "8080"),
			ReadTimeout:  getEnvAsDuration("SERVER_READ_TIMEOUT", "15s"),
			WriteTimeout: getEnvAsDuration("SERVER_WRITE_TIMEOUT", "15s"),
			IdleTimeout:  getEnvAsDuration("SERVER_IDLE_TIMEOUT", "60s"),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			User:            getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", "postgres"),
			DBName:          getEnv("DB_NAME", "mockbank"),
			SSLMode:         getEnv("DB_SSLMODE", "disable"),
			MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getEnvAsDuration("DB_CONN_MAX_LIFETIME", "5m"),
		},
		App: AppConfig{
			FailureRate:        getEnvAsFloat("FAILURE_RATE", 0.05),
			MinLatencyMS:       getEnvAsInt("MIN_LATENCY_MS", 100),
			MaxLatencyMS:       getEnvAsInt("MAX_LATENCY_MS", 2000),
			AuthExpiryHours:    authExpiryHours,
			AuthExpiryDuration: time.Duration(authExpiryHours) * time.Hour,
		},
		Logger: LoggerConfig{
			Level: getEnv("LOG_LEVEL", "info"),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.Server.Port == "" {
		return fmt.Errorf("server port cannot be empty")
	}

	if c.Database.Host == "" {
		return fmt.Errorf("database host cannot be empty")
	}
	if c.Database.DBName == "" {
		return fmt.Errorf("database name cannot be empty")
	}

	if c.App.FailureRate < 0 || c.App.FailureRate > 1 {
		return fmt.Errorf("failure rate must be between 0 and 1, got %f", c.App.FailureRate)
	}

	if c.App.MinLatencyMS < 0 {
		return fmt.Errorf("min latency cannot be negative")
	}
	if c.App.MaxLatencyMS < c.App.MinLatencyMS {
		return fmt.Errorf("max latency (%d) must be >= min latency (%d)", c.App.MaxLatencyMS, c.App.MinLatencyMS)
	}

	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[c.Logger.Level] {
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", c.Logger.Level)
	}

	return nil
}

// DSN returns the PostgreSQL connection string
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvAsFloat(key string, defaultValue float64) float64 {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvAsDuration(key, defaultValue string) time.Duration {
	valueStr := getEnv(key, defaultValue)
	duration, err := time.ParseDuration(valueStr)
	if err != nil {
		// Fallback to parsing the default if provided value is invalid
		duration, err = time.ParseDuration(defaultValue)
		if err != nil {
			return 0
		}
	}
	return duration
}
