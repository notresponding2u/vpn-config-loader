package config

import (
	"errors"
	"os"
)

type Config struct {
	OpenVPNAttributeManagementIP     string
	OpenVPNAttributeManagementPort   string
	OpenVPNConfigFile                string
	OpenVPNConfigFilesDir            string
	OpenVPNConfigFileDir             string
	OpenVPNConfigAuthFileAddition    string
}

func New() (*Config, error) {
	c := &Config{
		OpenVPNAttributeManagementIP:     "127.0.0.1",
		// For now hard coded.
		OpenVPNConfigAuthFileAddition:    "auth-user-pass ./auth.txt",
	}

	c.OpenVPNConfigFile = os.Getenv("VPN_CONFIG_FILE")
	if c.OpenVPNCommand == "" {
		return nil, errors.New("VPN_CONFIG_FILE env not set")
	}

	c.OpenVPNConfigFilesDir = os.Getenv("VPN_CONFIG_FILES_DIR")
	if c.OpenVPNCommand == "" {
		return nil, errors.New("VPN_CONFIG_FILES_DIR env not set")
	}

	c.OpenVPNConfigFileDir = os.Getenv("VPN_CONFIG_FILE_DIR")
	if c.OpenVPNCommand == "" {
		return nil, errors.New("VPN_CONFIG_FILE_DIR env not set")
	}

	c.OpenVPNAttributeManagementPort = os.Getenv("VPN_CONFIG_MANAGEMENT_PORT")
	if c.OpenVPNAttributeManagementPort == "" {
		return nil, errors.New("VPN_CONFIG_MANAGEMENT_PORT env not set")
	}

	return c, nil
}
