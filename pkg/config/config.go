// Package config provides unified configuration for both geo-mobile and solo modes
package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

// Mode represents deployment mode
type Mode string

const (
	ModeGeoMobile Mode = "geo-mobile"
	ModeSolo      Mode = "solo"
)

// Config holds all application configuration
type Config struct {
	// Deployment mode
	Mode Mode `json:"mode"`

	// Server configuration
	Server ServerConfig `json:"server"`

	// Database configuration
	Database DatabaseConfig `json:"database"`

	// Cache configuration
	Cache CacheConfig `json:"cache"`

	// Events configuration
	Events EventsConfig `json:"events"`

	// Feature flags
	Features FeatureConfig `json:"features"`

	// Logging
	Log LogConfig `json:"log"`
}

// ServerConfig holds HTTP server settings
type ServerConfig struct {
	Port         int    `json:"port"`
	Host         string `json:"host"`
	ReadTimeout  int    `json:"read_timeout_seconds"`
	WriteTimeout int    `json:"write_timeout_seconds"`
	MaxConnections int  `json:"max_connections"`
}

// DatabaseConfig holds database settings
type DatabaseConfig struct {
	Backend  string `json:"backend"` // "postgres" or "sqlite"
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database"`
	SSLMode  string `json:"ssl_mode"` // "disable", "require", etc.
	MaxConns int    `json:"max_connections"`
}

// CacheConfig holds cache settings
type CacheConfig struct {
	Backend    string `json:"backend"` // "memory" or "redis"
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Password   string `json:"password"`
	DefaultTTL int    `json:"default_ttl_seconds"`
}

// EventsConfig holds event bus settings
type EventsConfig struct {
	Backend string `json:"backend"` // "memory" or "kafka"
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Topic   string `json:"topic"`
}

// FeatureConfig holds feature flags
type FeatureConfig struct {
	EnableWebSocket bool `json:"enable_websocket"`
	EnableOCR       bool `json:"enable_ocr"`
	EnableSatLink   bool `json:"enable_satlink"`
	ReadOnly        bool `json:"read_only"`
}

// LogConfig holds logging settings
type LogConfig struct {
	Level  string `json:"level"` // "debug", "info", "warn", "error"
	Format string `json:"format"` // "json" or "text"
	Output string `json:"output"` // "stdout", "stderr", or file path
}

// LoadConfig loads configuration from environment variables and defaults
func LoadConfig() *Config {
	mode := parseMode(getEnv("CADASTRE_MODE", "solo"))

	config := &Config{
		Mode: mode,
		Server: ServerConfig{
			Port:           getEnvInt("PORT", 8080),
			Host:           getEnv("HOST", "0.0.0.0"),
			ReadTimeout:    getEnvInt("READ_TIMEOUT", 15),
			WriteTimeout:   getEnvInt("WRITE_TIMEOUT", 15),
			MaxConnections: getEnvInt("MAX_CONNECTIONS", 100),
		},
		Log: LogConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "text"),
			Output: getEnv("LOG_OUTPUT", "stdout"),
		},
	}

	// Mode-specific configuration
	if mode == ModeGeoMobile {
		config.loadGeoMobileConfig()
	} else {
		config.loadSoloConfig()
	}

	return config
}

// loadGeoMobileConfig configures for cloud deployment
func (c *Config) loadGeoMobileConfig() {
	c.Database = DatabaseConfig{
		Backend:  getEnv("DB_BACKEND", "postgres"),
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnvInt("DB_PORT", 5432),
		Username: getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", ""),
		Database: getEnv("DB_NAME", "cadastre_ia"),
		SSLMode:  getEnv("DB_SSLMODE", "require"),
		MaxConns: getEnvInt("DB_MAX_CONNS", 25),
	}

	c.Cache = CacheConfig{
		Backend:    getEnv("CACHE_BACKEND", "redis"),
		Host:       getEnv("CACHE_HOST", "localhost"),
		Port:       getEnvInt("CACHE_PORT", 6379),
		Password:   getEnv("CACHE_PASSWORD", ""),
		DefaultTTL: getEnvInt("CACHE_TTL", 3600),
	}

	c.Events = EventsConfig{
		Backend: getEnv("EVENTS_BACKEND", "kafka"),
		Host:    getEnv("EVENTS_HOST", "localhost"),
		Port:    getEnvInt("EVENTS_PORT", 9092),
		Topic:   getEnv("EVENTS_TOPIC", "cadastre-events"),
	}

	c.Features = FeatureConfig{
		EnableWebSocket: true,
		EnableOCR:       true,
		EnableSatLink:   true,
		ReadOnly:        false,
	}
}

// loadSoloConfig configures for local/desktop deployment
func (c *Config) loadSoloConfig() {
	dbPath := getEnv("DB_PATH", "./cadastre.db")

	c.Database = DatabaseConfig{
		Backend:  getEnv("DB_BACKEND", "sqlite"),
		Database: dbPath,
		MaxConns: 1, // SQLite doesn't support multiple connections well
	}

	c.Cache = CacheConfig{
		Backend:    getEnv("CACHE_BACKEND", "memory"),
		DefaultTTL: getEnvInt("CACHE_TTL", 1800),
	}

	c.Events = EventsConfig{
		Backend: getEnv("EVENTS_BACKEND", "memory"),
	}

	c.Features = FeatureConfig{
		EnableWebSocket: getEnvBool("ENABLE_WEBSOCKET", false),
		EnableOCR:       getEnvBool("ENABLE_OCR", true),
		EnableSatLink:   getEnvBool("ENABLE_SATLINK", false),
		ReadOnly:        false,
	}
}

// GetDatabaseURL returns the appropriate database connection string
func (c *Config) GetDatabaseURL() string {
	switch c.Database.Backend {
	case "postgres":
		sslMode := c.Database.SSLMode
		if sslMode == "" {
			sslMode = "disable"
		}
		return fmt.Sprintf(
			"postgres://%s:%s@%s:%d/%s?sslmode=%s",
			c.Database.Username,
			c.Database.Password,
			c.Database.Host,
			c.Database.Port,
			c.Database.Database,
			sslMode,
		)
	case "sqlite":
		return fmt.Sprintf("file:%s?cache=shared&mode=rwc", c.Database.Database)
	default:
		return ""
	}
}

// GetCacheURL returns cache connection URL
func (c *Config) GetCacheURL() string {
	if c.Cache.Backend != "redis" {
		return ""
	}
	return fmt.Sprintf(
		"%s:%d",
		c.Cache.Host,
		c.Cache.Port,
	)
}

// Validate checks configuration validity
func (c *Config) Validate() error {
	if c.Mode != ModeGeoMobile && c.Mode != ModeSolo {
		return fmt.Errorf("invalid mode: %s (must be geo-mobile or solo)", c.Mode)
	}

	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Server.Port)
	}

	if c.Log.Level != "debug" && c.Log.Level != "info" && c.Log.Level != "warn" && c.Log.Level != "error" {
		return fmt.Errorf("invalid log level: %s", c.Log.Level)
	}

	return nil
}

// LogConfig returns a logger instance
func (c *Config) GetLogger() *log.Logger {
	prefix := fmt.Sprintf("[cadastre-%s] ", c.Mode)
	return log.New(os.Stdout, prefix, log.LstdFlags|log.Lshortfile)
}

// Helper functions

func parseMode(s string) Mode {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "geo-mobile", "geomobile", "cloud":
		return ModeGeoMobile
	case "solo", "local", "desktop":
		return ModeSolo
	default:
		return ModeSolo // Default to solo
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		switch strings.ToLower(value) {
		case "true", "1", "yes":
			return true
		case "false", "0", "no":
			return false
		}
	}
	return defaultValue
}

// PrintConfig outputs configuration for verification
func (c *Config) Print(logger *log.Logger) {
	logger.Println("="*70)
	logger.Printf("CADASTRE CONFIGURATION\n")
	logger.Println("="*70)
	logger.Printf("Mode:               %s\n", c.Mode)
	logger.Printf("Server:             %s:%d\n", c.Server.Host, c.Server.Port)
	logger.Printf("Database:           %s\n", c.Database.Backend)
	if c.Database.Backend == "postgres" {
		logger.Printf("  - Host:           %s:%d\n", c.Database.Host, c.Database.Port)
		logger.Printf("  - Database:       %s\n", c.Database.Database)
	}
	logger.Printf("Cache:              %s\n", c.Cache.Backend)
	logger.Printf("Events:             %s\n", c.Events.Backend)
	logger.Printf("Log Level:          %s\n", c.Log.Level)
	logger.Printf("WebSocket:          %v\n", c.Features.EnableWebSocket)
	logger.Printf("OCR:                %v\n", c.Features.EnableOCR)
	logger.Println("="*70)
}
