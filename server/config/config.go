package config

import (
	"errors"
	"log/slog"
	"os"
)

type Config struct {
	OpenVPNAttributeManagementIP   string
	OpenVPNAttributeManagementPort string
	OpenVPNConfigFile              string
	OpenVPNConfigFilesDir          string
	OpenVPNConfigFileDir           string
	OpenVPNConfigAuthFileAddition  string
	LogLevel                       slog.Level
}

func New() (*Config, error) {
	c := &Config{
		OpenVPNAttributeManagementIP: "127.0.0.1",
		// For now hard coded.
		OpenVPNConfigAuthFileAddition: "auth-user-pass ./auth.txt",
	}

	c.OpenVPNConfigFile = os.Getenv("VPN_CONFIG_FILE")
	if c.OpenVPNConfigFile == "" {
		return nil, errors.New("VPN_CONFIG_FILE env not set")
	}

	c.OpenVPNConfigFilesDir = os.Getenv("VPN_CONFIG_FILES_DIR")
	if c.OpenVPNConfigFilesDir == "" {
		return nil, errors.New("VPN_CONFIG_FILES_DIR env not set")
	}

	c.OpenVPNConfigFileDir = os.Getenv("VPN_CONFIG_FILE_DIR")
	if c.OpenVPNConfigFileDir == "" {
		return nil, errors.New("VPN_CONFIG_FILE_DIR env not set")
	}

	c.OpenVPNAttributeManagementPort = os.Getenv("VPN_CONFIG_MANAGEMENT_PORT")
	if c.OpenVPNAttributeManagementPort == "" {
		return nil, errors.New("VPN_CONFIG_MANAGEMENT_PORT env not set")
	}

	logLevelEnv := os.Getenv("LOG_LEVEL")
	if logLevelEnv == "" {
		c.LogLevel = slog.LevelInfo
	} else {

		logLevelMap := map[string]slog.Level{
			"DEBUG":   slog.LevelDebug,
			"ERROR":   slog.LevelError,
			"INFO":    slog.LevelInfo,
			"WARNING": slog.LevelWarn,
		}

		level, ok := logLevelMap[logLevelEnv]
		if !ok {
			return nil, errors.New("unexpected log level")
		}

		c.LogLevel = level
	}

	return c, nil
}
