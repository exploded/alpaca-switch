// Package hikvision implements a SwitchBackend for Hikvision IP camera IR illuminators.
// Each CameraConfig entry becomes one switch (on = IR enabled, off = IR disabled).
// Hardware communication uses the Hikvision ISAPI over HTTP with Digest authentication.
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

// CameraConfig holds connection details for one Hikvision camera.
type CameraConfig struct {
	Host     string `json:"host"`
	Username string `json:"username"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

// camera is the runtime representation of one camera switch.
type camera struct {
	cfg        CameraConfig
	customName string
	client     *http.Client
	state      bool // cached state
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

// Connect queries current IR state from all cameras.
func (b *Backend) Connect() error {
	b.mu.Lock()
	b.connected = true
	b.mu.Unlock()
	// Best-effort: query initial state from each camera
	for i, cam := range b.cameras {
		on, err := cam.getIRLight()
		if err != nil {
			log.Printf("[hikvision] warning: could not query camera %d (%s): %v", i, cam.cfg.Host, err)
			continue
		}
		b.mu.Lock()
		b.cameras[i].state = on
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
	if b.cameras[id].customName != "" {
		return b.cameras[id].customName
	}
	return b.cameras[id].cfg.Name
}

// SetName sets a custom name for switch id.
func (b *Backend) SetName(id int, name string) error {
	if id < 0 || id >= len(b.cameras) {
		return fmt.Errorf("invalid camera id %d", id)
	}
	b.mu.Lock()
	b.cameras[id].customName = name
	b.mu.Unlock()
	return nil
}

// GetDescription returns a description for switch id.
func (b *Backend) GetDescription(id int) string {
	return fmt.Sprintf("%s IR illuminator", b.GetName(id))
}

// GetCanWrite reports whether switch id is writable (always true for IR).
func (b *Backend) GetCanWrite(id int) bool { return true }

// GetMin returns the minimum value (0 = off).
func (b *Backend) GetMin(id int) float64 { return 0 }

// GetMax returns the maximum value (1 = on).
func (b *Backend) GetMax(id int) float64 { return 1 }

// GetStep returns the step size (1).
func (b *Backend) GetStep(id int) float64 { return 1 }

// GetSwitch returns the cached on/off state of switch id.
func (b *Backend) GetSwitch(id int) (bool, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if id < 0 || id >= len(b.cameras) {
		return false, fmt.Errorf("invalid camera id %d", id)
	}
	return b.cameras[id].state, nil
}

// GetSwitchValue returns the numeric value of switch id (0.0 or 1.0).
func (b *Backend) GetSwitchValue(id int) (float64, error) {
	on, err := b.GetSwitch(id)
	if err != nil {
		return 0, err
	}
	if on {
		return 1, nil
	}
	return 0, nil
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
	b.cameras[id].state = state
	b.mu.Unlock()
	log.Printf("[hikvision] camera %d (%s) IR set to %v", id, cam.cfg.Name, state)
	return nil
}

// SetSwitchValue sets the IR illuminator by numeric value.
func (b *Backend) SetSwitchValue(id int, value float64) error {
	return b.SetSwitch(id, value != 0)
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
