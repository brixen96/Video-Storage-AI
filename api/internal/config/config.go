package config

import (
	"encoding/json"
	"fmt"
	"log"
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

// AppUserConfig holds user-modifiable settings persisted to config.json in AppData.
// These are the defaults; environment variables always take precedence.
type AppUserConfig struct {
	ServerPort string `json:"server_port"`
	WebPort    string `json:"web_port"`
}

// dataDir is the resolved application data directory, set once in Load().
var dataDir string

// userConfig is the in-memory cache of the persisted user config.
var userConfig *AppUserConfig

// DataDir returns the application data directory.
// On Windows this is %APPDATA%\Video Storage AI.
// On other platforms it falls back to $HOME/.video-storage-ai.
func DataDir() string {
	return dataDir
}

// GetUserConfig returns the current persisted user configuration.
func GetUserConfig() AppUserConfig {
	if userConfig == nil {
		return AppUserConfig{ServerPort: "8080"}
	}
	return *userConfig
}

// SaveUserConfig persists the user configuration to config.json.
func SaveUserConfig(cfg AppUserConfig) error {
	if cfg.ServerPort == "" {
		cfg.ServerPort = "8080"
	}
	if cfg.WebPort == "" {
		cfg.WebPort = "3000"
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	configPath := filepath.Join(dataDir, "config.json")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config.json: %w", err)
	}
	userConfig = &cfg
	return nil
}

// resolveDataDir computes the canonical application data directory.
func resolveDataDir() string {
	// Windows: %APPDATA%\Video Storage AI
	if appData := os.Getenv("APPDATA"); appData != "" {
		return filepath.Join(appData, "Video Storage AI")
	}
	// macOS / Linux fallback
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".video-storage-ai")
	}
	return filepath.Join(".", "data")
}

// loadUserConfig reads config.json from the data directory, creating it with defaults if missing.
func loadUserConfig() AppUserConfig {
	defaults := AppUserConfig{ServerPort: "8080", WebPort: "3000"}

	configPath := filepath.Join(dataDir, "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		// File doesn't exist yet — write defaults so the user can find and edit it
		if writeData, jsonErr := json.MarshalIndent(defaults, "", "  "); jsonErr == nil {
			if writeErr := os.WriteFile(configPath, writeData, 0644); writeErr != nil {
				log.Printf("Warning: could not write default config.json: %v", writeErr)
			}
		}
		return defaults
	}

	var cfg AppUserConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Printf("Warning: config.json is invalid, using defaults: %v", err)
		return defaults
	}
	if cfg.ServerPort == "" {
		cfg.ServerPort = defaults.ServerPort
	}
	if cfg.WebPort == "" {
		cfg.WebPort = defaults.WebPort
	}

	return cfg
}

// Load reads configuration from environment variables and .env file.
func Load() (*Config, error) {
	// Resolve (and cache) the data directory first so defaults below can use it.
	if dataDir == "" {
		dataDir = resolveDataDir()
	}

	// Ensure the top-level data directory exists.
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory %s: %w", dataDir, err)
	}

	// Load user config from config.json (port and other persisted settings).
	userCfg := loadUserConfig()
	userConfig = &userCfg

	// Try to load .env from the executable's directory, then cwd.
	exePath, _ := os.Executable()
	_ = godotenv.Load(filepath.Join(filepath.Dir(exePath), ".env"))
	_ = godotenv.Load(".env")

	config := &Config{
		Server: ServerConfig{
			// env var takes precedence over config.json, config.json over hardcoded default
			Port:         getEnv("SERVER_PORT", userCfg.ServerPort),
			Host:         getEnv("SERVER_HOST", "localhost"),
			Mode:         getEnv("SERVER_MODE", "release"),
			ReadTimeout:  getEnvAsInt("SERVER_READ_TIMEOUT", 15),
			WriteTimeout: getEnvAsInt("SERVER_WRITE_TIMEOUT", 600), // 10 minutes — SSE streams need generous write timeout
		},
		Database: DatabaseConfig{
			Path:        getEnv("DATABASE_PATH", filepath.Join(dataDir, "videos.db")),
			MaxIdleConn: getEnvAsInt("DB_MAX_IDLE_CONN", 10),
			MaxOpenConn: getEnvAsInt("DB_MAX_OPEN_CONN", 100),
		},
		Paths: PathsConfig{
			ThumbnailDir:  getEnv("THUMBNAIL_DIR", filepath.Join(dataDir, "thumbnails")),
			PerformerDir:  getEnv("PERFORMER_DIR", filepath.Join(dataDir, "performers")),
			AssetsBaseDir: getEnv("ASSETS_BASE_DIR", dataDir),
		},
		API: APIConfig{
			AdultDataLinkAPIKey: getEnv("ADULTDATALINK_API_KEY", ""),
		},
	}

	return config, nil
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
