package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	WebPort                  string `json:"web_port"`
	Language                 string `json:"language"`
	WatchInterval            int    `json:"watch_interval"`
	Units                    string `json:"units"`
	NoColor                  bool   `json:"no_color"`
	ProcessesSortBy          string `json:"processes_sort_by"`
	BackgroundNetworkHistory bool   `json:"background_network_history"`
}

func loadConfig() *Config {
	// Default values
	cfg := &Config{
		WebPort:                  "8080",
		Language:                 "ru",
		WatchInterval:            2,
		Units:                    "auto",
		NoColor:                  false,
		ProcessesSortBy:          "cpu",
		BackgroundNetworkHistory: false,
	}

	exePath, err := os.Executable()
	if err != nil {
		return cfg
	}

	configPath := filepath.Join(filepath.Dir(exePath), "sysinfogo_config.json")
	data, err := os.ReadFile(configPath)
	if err == nil {
		_ = json.Unmarshal(data, cfg)
	}

	return cfg
}

func SaveDefaultConfig(path string) error {
	cfg := &Config{
		WebPort:                  "8080",
		Language:                 "ru",
		WatchInterval:            2,
		Units:                    "auto",
		NoColor:                  false,
		ProcessesSortBy:          "cpu",
		BackgroundNetworkHistory: false,
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
