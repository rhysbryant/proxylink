package config

import (
	"fmt"
	"os"

	"github.com/rhysbryant/proxylink/pkg/rulesengine"
	"gopkg.in/yaml.v2"
)

type Config struct {
	ListenAddr string             `yaml:"listen"` // Address to listen on
	Mode       string             `yaml:"mode"`   // standalone, bridge, exit
	TLS        TLSConfig          `yaml:"tls"`    // TLS configuration
	Rules      []rulesengine.Rule `yaml:"rules"`  // Proxy rules
	Key        string             `yaml:"wsKey"`  // 32-byte key for encrypting WebSocket traffic (optional)
}

type TLSConfig struct {
	CertFile    string `yaml:"cert"`        // Path to TLS certificate file
	KeyFile     string `yaml:"key"`         // Path to TLS key file
	LetsEncrypt bool   `yaml:"letsEncrypt"` // Enable Let's Encrypt support
	Domain      string `yaml:"domain"`      // Domain name for Let's Encrypt
}

func LoadConfig(filePath string) (*Config, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	var cfg Config
	if err := yaml.NewDecoder(file).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}
