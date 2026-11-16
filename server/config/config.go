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
	WebPort                        string
	LogLevel                       slog.Level
}

func New() (*Config, error) {
	c := &Config{
		OpenVPNAttributeManagementIP:   "127.0.0.1",
		OpenVPNAttributeManagementPort: "6061",
		OpenVPNConfigFilesDir:          "/app/conf.d",
	}

	c.OpenVPNConfigFile = os.Getenv("VPN_CONFIG_FILE")
	if c.OpenVPNConfigFile == "" {
		return nil, errors.New("VPN_CONFIG_FILE env not set")
	}

	c.OpenVPNConfigFileDir = os.Getenv("VPN_CONFIG_FILE_DIR")
	if c.OpenVPNConfigFileDir == "" {
		return nil, errors.New("VPN_CONFIG_FILE_DIR env not set")
	}

	c.WebPort = os.Getenv("WEB_PORT")
	if c.WebPort == "" {
		return nil, errors.New("WEB_PORT env not set")
	}

	logLevelEnv := os.Getenv("LOG_LEVEL")
	if logLevelEnv == "" {
		c.LogLevel = slog.LevelError
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
