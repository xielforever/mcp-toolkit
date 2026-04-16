package config

import (
	"errors"
	"os"
	"strings"
)

type Config struct {
	ListenAddr         string
	BasePath           string
	ServiceToken       string
	VCenterURL         string
	VCenterUsername    string
	VCenterPassword    string
	VCenterInsecure    bool
	VCenterCAFile      string
	EnableDangerousOps bool
}

func Load() (Config, error) {
	c := Config{
		ListenAddr:      os.Getenv("HTTP_LISTEN_ADDR"),
		BasePath:        os.Getenv("MCP_HTTP_BASE_PATH"),
		ServiceToken:    os.Getenv("MCP_SERVICE_TOKEN"),
		VCenterURL:      os.Getenv("VCENTER_URL"),
		VCenterUsername: os.Getenv("VCENTER_USERNAME"),
		VCenterPassword: os.Getenv("VCENTER_PASSWORD"),
		VCenterCAFile:   os.Getenv("VCENTER_CA_FILE"),
	}

	if c.ListenAddr == "" {
		c.ListenAddr = "0.0.0.0:8080"
	}

	c.VCenterInsecure = strings.EqualFold(os.Getenv("VCENTER_INSECURE"), "true")
	c.EnableDangerousOps = strings.EqualFold(os.Getenv("VCENTER_ENABLE_DANGEROUS_OPS"), "true")

	if c.ServiceToken == "" {
		return Config{}, errors.New("MCP_SERVICE_TOKEN is required")
	}
	if c.VCenterURL == "" || c.VCenterUsername == "" || c.VCenterPassword == "" {
		return Config{}, errors.New("VCENTER_URL/VCENTER_USERNAME/VCENTER_PASSWORD are required")
	}

	return c, nil
}

