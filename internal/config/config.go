package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server  ServerConfig  `yaml:"server"`
	Tunnels []TunnelConfig `yaml:"tunnels"`
}

type ServerConfig struct {
	ListenAddr  string `yaml:"listen_addr"`  // Web UI
	ControlAddr string `yaml:"control_addr"` // Tunnel control (Yamux)
	PublicAddr  string `yaml:"public_addr"`
	Secret      string `yaml:"secret"`
}

type TunnelConfig struct {
	ID           string `yaml:"id"`
	Name         string `yaml:"name"`
	ExternalPort int    `yaml:"external_port"`
	InternalPort int    `yaml:"internal_port"`
	Type         string `yaml:"type"` // "tcp", "udp", "both"
	Enabled      bool   `yaml:"enabled"`
}

func Load(path string) (*Config, error) {
	if path == "" {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, ".suptunnels", "config.yaml")
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{
				Server: ServerConfig{
					ListenAddr: ":8080",
					Secret:     "supersecret",
				},
				Tunnels: []TunnelConfig{},
			}, nil
		}
		return nil, err
	}
	defer f.Close()

	var cfg Config
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) Save(path string) error {
	if path == "" {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, ".suptunnels", "config.yaml")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return yaml.NewEncoder(f).Encode(c)
}
