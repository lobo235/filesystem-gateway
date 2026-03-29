package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// AllowedHost represents an allowed download host with an optional path prefix.
type AllowedHost struct {
	Host       string
	PathPrefix string // empty means any path on this host is allowed
}

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	NFSBasePath          string
	GatewayAPIKey        string
	Port                 string
	LogLevel             string
	DataDir              string
	MaxDownloadSize      int64
	MaxWriteFileSize     int64
	MaxExtractSize       int64
	AllowedDownloadHosts []AllowedHost
}

// Load reads configuration from environment variables, applying defaults and validating required fields.
func Load() (*Config, error) {
	// Load .env if present — ignore error if file doesn't exist.
	_ = godotenv.Load()

	cfg := &Config{
		NFSBasePath:   os.Getenv("NFS_BASE_PATH"),
		GatewayAPIKey: os.Getenv("GATEWAY_API_KEY"),
		Port:          os.Getenv("PORT"),
		LogLevel:      os.Getenv("LOG_LEVEL"),
		DataDir:       os.Getenv("DATA_DIR"),
	}

	if cfg.NFSBasePath == "" {
		return nil, fmt.Errorf("NFS_BASE_PATH is required")
	}
	if cfg.GatewayAPIKey == "" {
		return nil, fmt.Errorf("GATEWAY_API_KEY is required")
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	if cfg.DataDir == "" {
		cfg.DataDir = "/data"
	}
	switch cfg.LogLevel {
	case "debug", "info", "warn", "error":
		// valid
	case "":
		cfg.LogLevel = "info"
	default:
		return nil, fmt.Errorf("LOG_LEVEL must be one of: debug, info, warn, error")
	}

	// Parse MAX_DOWNLOAD_SIZE (default 2GB).
	maxDL := os.Getenv("MAX_DOWNLOAD_SIZE")
	if maxDL == "" {
		cfg.MaxDownloadSize = 2147483648 // 2GB
	} else {
		v, err := strconv.ParseInt(maxDL, 10, 64)
		if err != nil || v <= 0 {
			return nil, fmt.Errorf("MAX_DOWNLOAD_SIZE must be a positive integer")
		}
		cfg.MaxDownloadSize = v
	}

	// Parse MAX_WRITE_FILE_SIZE (default 1MB).
	maxWF := os.Getenv("MAX_WRITE_FILE_SIZE")
	if maxWF == "" {
		cfg.MaxWriteFileSize = 1048576 // 1MB
	} else {
		v, err := strconv.ParseInt(maxWF, 10, 64)
		if err != nil || v <= 0 {
			return nil, fmt.Errorf("MAX_WRITE_FILE_SIZE must be a positive integer")
		}
		cfg.MaxWriteFileSize = v
	}

	// Parse MAX_EXTRACT_SIZE (default 10GB).
	maxES := os.Getenv("MAX_EXTRACT_SIZE")
	if maxES == "" {
		cfg.MaxExtractSize = 10737418240 // 10GB
	} else {
		v, err := strconv.ParseInt(maxES, 10, 64)
		if err != nil || v <= 0 {
			return nil, fmt.Errorf("MAX_EXTRACT_SIZE must be a positive integer")
		}
		cfg.MaxExtractSize = v
	}

	// Parse ALLOWED_DOWNLOAD_HOSTS (comma-separated, host or host/pathPrefix).
	hostsEnv := os.Getenv("ALLOWED_DOWNLOAD_HOSTS")
	if hostsEnv != "" {
		for _, entry := range strings.Split(hostsEnv, ",") {
			entry = strings.TrimSpace(entry)
			if entry == "" {
				continue
			}
			// Split on first / to separate host from path prefix.
			if idx := strings.Index(entry, "/"); idx >= 0 {
				cfg.AllowedDownloadHosts = append(cfg.AllowedDownloadHosts, AllowedHost{
					Host:       entry[:idx],
					PathPrefix: "/" + strings.TrimLeft(entry[idx:], "/"),
				})
			} else {
				cfg.AllowedDownloadHosts = append(cfg.AllowedDownloadHosts, AllowedHost{
					Host: entry,
				})
			}
		}
	}

	return cfg, nil
}
