package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"alpaca-switch/backend"
	"alpaca-switch/backend/hikvision"
	"alpaca-switch/backend/mi"
	"alpaca-switch/server"
)

// Config is the unified configuration file format.
type Config struct {
	AlpacaPort       int                    `json:"alpaca_port"`
	MiDevices        []mi.Device            `json:"mi_devices"`
	HikvisionCameras []hikvision.CameraConfig `json:"hikvision_cameras"`
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if cfg.AlpacaPort == 0 {
		cfg.AlpacaPort = 11111
	}
	return &cfg, nil
}

func main() {
	cfg, err := loadConfig("config/settings.json")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Build backends
	miBackend := mi.New(cfg.MiDevices, "")
	hikBackend := hikvision.New(cfg.HikvisionCameras)

	// Build router (Mi switches first, then Hikvision)
	router := backend.NewRouter([]backend.SwitchBackend{miBackend, hikBackend})

	log.Printf("alpaca-switch starting: %d total switches (%d Mi + %d Hikvision)",
		router.NumSwitches(), miBackend.NumSwitches(), hikBackend.NumSwitches())

	// Start discovery and API
	go server.StartDiscovery(32227, cfg.AlpacaPort)
	srv := server.New(router)
	srv.Start(fmt.Sprintf(":%d", cfg.AlpacaPort))
}
