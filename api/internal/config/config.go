package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Paths    PathsConfig
	API      APIConfig
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Port         string
	Host         string
	Mode         string // debug, release, test
	ReadTimeout  int
	WriteTimeout int
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Path        string
	MaxIdleConn int
	MaxOpenConn int
}

// PathsConfig holds filesystem paths
type PathsConfig struct {
	ThumbnailDir  string
	PerformerDir  string
	AssetsBaseDir string
}

// APIConfig holds external API configuration
type APIConfig struct {
	AdultDataLinkAPIKey string
}

// Load reads configuration from environment variables and .env file
func Load() (*Config, error) {
	// Get the executable directory for .env path
	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}
	exeDir := filepath.Dir(exePath)

	// Try to load .env file (don't fail if it doesn't exist)
	envPath := filepath.Join(exeDir, ".env")
	_ = godotenv.Load(envPath)

	// Also try loading from current working directory
	_ = godotenv.Load(".env")

	config := &Config{
		Server: ServerConfig{
			Port:         getEnv("SERVER_PORT", "8080"),
			Host:         getEnv("SERVER_HOST", "localhost"),
			Mode:         getEnv("SERVER_MODE", "error"),
			ReadTimeout:  getEnvAsInt("SERVER_READ_TIMEOUT", 15),
			WriteTimeout: getEnvAsInt("SERVER_WRITE_TIMEOUT", 15),
		},
		Database: DatabaseConfig{
			Path:        getEnv("DATABASE_PATH", "./data/video_storage.db"),
			MaxIdleConn: getEnvAsInt("DB_MAX_IDLE_CONN", 10),
			MaxOpenConn: getEnvAsInt("DB_MAX_OPEN_CONN", 100),
		},
		Paths: PathsConfig{
			ThumbnailDir:  getEnv("THUMBNAIL_DIR", "./assets/thumbnails"),
			PerformerDir:  getEnv("PERFORMER_DIR", "./assets/performers"),
			AssetsBaseDir: getEnv("ASSETS_BASE_DIR", "./assets"),
		},
		API: APIConfig{
			AdultDataLinkAPIKey: getEnv("ADULTDATALINK_API_KEY", ""),
		},
	}

	// Validate required fields
	if config.API.AdultDataLinkAPIKey == "" {
		return nil, fmt.Errorf("ADULTDATALINK_API_KEY is required")
	}

	return config, nil
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt retrieves an environment variable as int or returns a default value
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