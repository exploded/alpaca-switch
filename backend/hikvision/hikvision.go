// Package hikvision implements a SwitchBackend for Hikvision IP camera IR illuminators.
// Each CameraConfig entry becomes one switch (on = IR enabled, off = IR disabled).
// Hardware communication uses the Hikvision ISAPI over HTTP with Digest authentication.
//
// Camera requirements:
//   - Configuration → System → Maintenance → System Service → Hardware must be enabled
//   - Tested on: DS-2CD2343G0-I, DS-2CD2335-I
//
// Host field supports an optional port, e.g. "192.168.1.3:65005".
package hikvision

import (
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/icholy/digest"
)

// CameraConfig holds connection details and cached state for one Hikvision camera.
type CameraConfig struct {
	Host        string  `json:"host"`
	Username    string  `json:"username"`
	Password    string  `json:"password"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	UniqueID    string  `json:"uniqueid"`
	Value       float64 `json:"value"` // cached last-known state: 0=off, 1=on
}

// camera is the runtime representation of one camera switch.
type camera struct {
	cfg    CameraConfig
	client *http.Client
}

// Backend implements backend.SwitchBackend for Hikvision IR switches.
type Backend struct {
	mu        sync.RWMutex
	cameras   []*camera
	connected bool
}

// New creates a Hikvision backend from a list of camera configs.
func New(cfgs []CameraConfig) *Backend {
	cams := make([]*camera, len(cfgs))
	for i, cfg := range cfgs {
		cams[i] = &camera{
			cfg: cfg,
			client: &http.Client{
				Transport: &digest.Transport{
					Username: cfg.Username,
					Password: cfg.Password,
				},
			},
		}
	}
	return &Backend{cameras: cams}
}

// Connect queries current IR state from all cameras and marks the backend connected.
func (b *Backend) Connect() error {
	b.mu.Lock()
	b.connected = true
	b.mu.Unlock()
	for i, cam := range b.cameras {
		on, err := cam.getIRLight()
		if err != nil {
			log.Printf("[hikvision] warning: could not query camera %d (%s): %v", i, cam.cfg.Host, err)
			continue
		}
		b.mu.Lock()
		if on {
			b.cameras[i].cfg.Value = 1
		} else {
			b.cameras[i].cfg.Value = 0
		}
		b.mu.Unlock()
		log.Printf("[hikvision] camera %d (%s) IR: %v", i, cam.cfg.Name, on)
	}
	return nil
}

// Disconnect marks the backend disconnected.
func (b *Backend) Disconnect() {
	b.mu.Lock()
	b.connected = false
	b.mu.Unlock()
}

// IsConnected reports whether the backend is connected.
func (b *Backend) IsConnected() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.connected
}

// NumSwitches returns the number of cameras (one switch per camera).
func (b *Backend) NumSwitches() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.cameras)
}

// GetName returns the camera name for switch id.
func (b *Backend) GetName(id int) string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if id < 0 || id >= len(b.cameras) {
		return ""
	}
	return b.cameras[id].cfg.Name
}

// SetName sets a custom name for switch id (persisted via the config layer).
func (b *Backend) SetName(id int, name string) error {
	if id < 0 || id >= len(b.cameras) {
		return fmt.Errorf("invalid camera id %d", id)
	}
	b.mu.Lock()
	b.cameras[id].cfg.Name = name
	b.mu.Unlock()
	return nil
}

// GetDescription returns the description for switch id.
// If no description is set in config, falls back to "<name> IR illuminator".
func (b *Backend) GetDescription(id int) string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if id < 0 || id >= len(b.cameras) {
		return ""
	}
	if b.cameras[id].cfg.Description != "" {
		return b.cameras[id].cfg.Description
	}
	return fmt.Sprintf("%s IR illuminator", b.cameras[id].cfg.Name)
}

// GetCanWrite always returns true — IR illuminators are always writable.
func (b *Backend) GetCanWrite(_ int) bool { return true }

// GetMin returns the minimum value (0 = off).
func (b *Backend) GetMin(_ int) float64 { return 0 }

// GetMax returns the maximum value (1 = on).
func (b *Backend) GetMax(_ int) float64 { return 1 }

// GetStep returns the step size (1).
func (b *Backend) GetStep(_ int) float64 { return 1 }

// GetSwitch queries the live IR state from the camera. The result is also
// cached in cfg.Value so GetSwitchValue stays consistent.
func (b *Backend) GetSwitch(id int) (bool, error) {
	b.mu.RLock()
	if id < 0 || id >= len(b.cameras) {
		b.mu.RUnlock()
		return false, fmt.Errorf("invalid camera id %d", id)
	}
	cam := b.cameras[id]
	b.mu.RUnlock()

	on, err := cam.getIRLight()
	if err != nil {
		return false, err
	}
	// Update cached value
	b.mu.Lock()
	if on {
		b.cameras[id].cfg.Value = 1
	} else {
		b.cameras[id].cfg.Value = 0
	}
	b.mu.Unlock()
	return on, nil
}

// GetSwitchValue returns the cached numeric value (0.0 or 1.0).
func (b *Backend) GetSwitchValue(id int) (float64, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if id < 0 || id >= len(b.cameras) {
		return 0, fmt.Errorf("invalid camera id %d", id)
	}
	return b.cameras[id].cfg.Value, nil
}

// SetSwitch turns the IR illuminator for switch id on or off.
func (b *Backend) SetSwitch(id int, state bool) error {
	b.mu.RLock()
	if id < 0 || id >= len(b.cameras) {
		b.mu.RUnlock()
		return fmt.Errorf("invalid camera id %d", id)
	}
	cam := b.cameras[id]
	b.mu.RUnlock()

	if err := cam.setIRLight(state); err != nil {
		return err
	}
	b.mu.Lock()
	if state {
		b.cameras[id].cfg.Value = 1
	} else {
		b.cameras[id].cfg.Value = 0
	}
	b.mu.Unlock()
	log.Printf("[hikvision] camera %d (%s) IR set to %v", id, cam.cfg.Name, state)
	return nil
}

// SetSwitchValue sets the IR illuminator by numeric value (0 = off, non-zero = on).
func (b *Backend) SetSwitchValue(id int, value float64) error {
	return b.SetSwitch(id, value != 0)
}

// Configs returns a snapshot of all camera configs (for config persistence).
func (b *Backend) Configs() []CameraConfig {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]CameraConfig, len(b.cameras))
	for i, c := range b.cameras {
		out[i] = c.cfg
	}
	return out
}

// ---------- low-level ISAPI calls ----------

// hardwareService is the XML envelope for /ISAPI/System/Hardware.
type hardwareService struct {
	XMLName       xml.Name      `xml:"HardwareService"`
	IrLightSwitch irLightSwitch `xml:"IrLightSwitch"`
}

type irLightSwitch struct {
	Mode string `xml:"mode"`
}

func (c *camera) setIRLight(on bool) error {
	mode := "close"
	if on {
		mode = "open"
	}
	payload, err := xml.Marshal(hardwareService{IrLightSwitch: irLightSwitch{Mode: mode}})
	if err != nil {
		return fmt.Errorf("marshal xml: %w", err)
	}
	url := fmt.Sprintf("http://%s/ISAPI/System/Hardware", c.cfg.Host)
	req, err := http.NewRequest(http.MethodPut, url, strings.NewReader(xml.Header+string(payload)))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/xml")
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("PUT %s: %w", url, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("camera returned %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (c *camera) getIRLight() (bool, error) {
	url := fmt.Sprintf("http://%s/ISAPI/System/Hardware", c.cfg.Host)
	resp, err := c.client.Get(url)
	if err != nil {
		return false, fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("camera returned %d: %s", resp.StatusCode, string(body))
	}
	var result hardwareService
	if err := xml.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("decode response: %w", err)
	}
	return result.IrLightSwitch.Mode == "open", nil
}
