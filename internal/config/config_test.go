package config

import (
	"os"
	"testing"
)

func clearEnv() {
	for _, key := range []string{
		"NFS_BASE_PATH", "GATEWAY_API_KEY",
		"PORT", "LOG_LEVEL", "DATA_DIR", "MAX_DOWNLOAD_SIZE", "MAX_WRITE_FILE_SIZE",
		"MAX_EXTRACT_SIZE", "ALLOWED_DOWNLOAD_HOSTS",
	} {
		os.Unsetenv(key)
	}
}

func setRequiredEnv() {
	os.Setenv("NFS_BASE_PATH", "/mnt/data/minecraft")
	os.Setenv("GATEWAY_API_KEY", "test-api-key")
}

func TestLoadSuccess(t *testing.T) {
	clearEnv()
	setRequiredEnv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.NFSBasePath != "/mnt/data/minecraft" {
		t.Errorf("NFSBasePath = %q, want /mnt/data/minecraft", cfg.NFSBasePath)
	}
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want 8080", cfg.Port)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want info", cfg.LogLevel)
	}
	if cfg.DataDir != "/data" {
		t.Errorf("DataDir = %q, want /data", cfg.DataDir)
	}
	if cfg.MaxDownloadSize != 2147483648 {
		t.Errorf("MaxDownloadSize = %d, want 2147483648", cfg.MaxDownloadSize)
	}
	if cfg.MaxWriteFileSize != 1048576 {
		t.Errorf("MaxWriteFileSize = %d, want 1048576", cfg.MaxWriteFileSize)
	}
	if cfg.MaxExtractSize != 10737418240 {
		t.Errorf("MaxExtractSize = %d, want 10737418240", cfg.MaxExtractSize)
	}
}

func TestLoadMissingRequired(t *testing.T) {
	tests := []struct {
		name   string
		unset  string
		errMsg string
	}{
		{"missing NFS_BASE_PATH", "NFS_BASE_PATH", "NFS_BASE_PATH is required"},
		{"missing GATEWAY_API_KEY", "GATEWAY_API_KEY", "GATEWAY_API_KEY is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv()
			setRequiredEnv()
			os.Unsetenv(tt.unset)

			_, err := Load()
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if err.Error() != tt.errMsg {
				t.Errorf("error = %q, want %q", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestLoadInvalidLogLevel(t *testing.T) {
	clearEnv()
	setRequiredEnv()
	os.Setenv("LOG_LEVEL", "invalid")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid LOG_LEVEL")
	}
}

func TestLoadCustomValues(t *testing.T) {
	clearEnv()
	setRequiredEnv()
	os.Setenv("PORT", "9090")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("DATA_DIR", "/custom/data")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != "9090" {
		t.Errorf("Port = %q, want 9090", cfg.Port)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want debug", cfg.LogLevel)
	}
	if cfg.DataDir != "/custom/data" {
		t.Errorf("DataDir = %q, want /custom/data", cfg.DataDir)
	}
}

func TestLoadCustomMaxDownloadSize(t *testing.T) {
	clearEnv()
	setRequiredEnv()
	os.Setenv("MAX_DOWNLOAD_SIZE", "4294967296")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.MaxDownloadSize != 4294967296 {
		t.Errorf("MaxDownloadSize = %d, want 4294967296", cfg.MaxDownloadSize)
	}
}

func TestLoadCustomMaxWriteFileSize(t *testing.T) {
	clearEnv()
	setRequiredEnv()
	os.Setenv("MAX_WRITE_FILE_SIZE", "2097152")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.MaxWriteFileSize != 2097152 {
		t.Errorf("MaxWriteFileSize = %d, want 2097152", cfg.MaxWriteFileSize)
	}
}

func TestLoadInvalidMaxDownloadSize(t *testing.T) {
	clearEnv()
	setRequiredEnv()
	os.Setenv("MAX_DOWNLOAD_SIZE", "not-a-number")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid MAX_DOWNLOAD_SIZE")
	}
}

func TestLoadInvalidMaxWriteFileSize(t *testing.T) {
	clearEnv()
	setRequiredEnv()
	os.Setenv("MAX_WRITE_FILE_SIZE", "not-a-number")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid MAX_WRITE_FILE_SIZE")
	}
}

func TestLoadZeroMaxDownloadSize(t *testing.T) {
	clearEnv()
	setRequiredEnv()
	os.Setenv("MAX_DOWNLOAD_SIZE", "0")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for zero MAX_DOWNLOAD_SIZE")
	}
}

func TestLoadNegativeMaxWriteFileSize(t *testing.T) {
	clearEnv()
	setRequiredEnv()
	os.Setenv("MAX_WRITE_FILE_SIZE", "-1")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for negative MAX_WRITE_FILE_SIZE")
	}
}

func TestLoadAllowedDownloadHosts(t *testing.T) {
	clearEnv()
	setRequiredEnv()
	os.Setenv("ALLOWED_DOWNLOAD_HOSTS", "custom.example.com,cdn.example.com/downloads/")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.AllowedDownloadHosts) != 2 {
		t.Fatalf("expected 2 hosts, got %d", len(cfg.AllowedDownloadHosts))
	}
	if cfg.AllowedDownloadHosts[0].Host != "custom.example.com" {
		t.Errorf("host[0] = %q, want custom.example.com", cfg.AllowedDownloadHosts[0].Host)
	}
	if cfg.AllowedDownloadHosts[0].PathPrefix != "" {
		t.Errorf("host[0] pathPrefix = %q, want empty", cfg.AllowedDownloadHosts[0].PathPrefix)
	}
	if cfg.AllowedDownloadHosts[1].Host != "cdn.example.com" {
		t.Errorf("host[1] = %q, want cdn.example.com", cfg.AllowedDownloadHosts[1].Host)
	}
	if cfg.AllowedDownloadHosts[1].PathPrefix != "/downloads/" {
		t.Errorf("host[1] pathPrefix = %q, want /downloads/", cfg.AllowedDownloadHosts[1].PathPrefix)
	}
}

func TestLoadAllowedDownloadHostsEmpty(t *testing.T) {
	clearEnv()
	setRequiredEnv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.AllowedDownloadHosts) != 0 {
		t.Errorf("expected 0 hosts, got %d", len(cfg.AllowedDownloadHosts))
	}
}

func TestLoadInvalidMaxExtractSize(t *testing.T) {
	clearEnv()
	setRequiredEnv()
	os.Setenv("MAX_EXTRACT_SIZE", "not-a-number")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid MAX_EXTRACT_SIZE")
	}
}
