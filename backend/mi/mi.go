package mi

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
)

// Device holds configuration and state for one Mi smart plug.
type Device struct {
	IP          string `json:"ip"`
	Token       string `json:"token"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Min         int64  `json:"min"`
	Max         int64  `json:"max"`
	Step        int64  `json:"step"`
	Canwrite    bool   `json:"canwrite"`
	Value       int64  `json:"value"`
}

// Backend implements backend.SwitchBackend for Xiaomi Mi smart plugs.
type Backend struct {
	mu         sync.RWMutex
	devices    []Device
	connected  bool
	savePath   string
	deviceLock []sync.Mutex // per-device operation lock
}

// New creates a Mi backend from a slice of device configs.
// savePath is the JSON file to persist state to (may be empty to skip persistence).
func New(devices []Device, savePath string) *Backend {
	return &Backend{
		devices:    devices,
		savePath:   savePath,
		deviceLock: make([]sync.Mutex, len(devices)),
	}
}

// Connect marks the backend connected and kicks off a background state refresh.
func (b *Backend) Connect() error {
	b.mu.Lock()
	b.connected = true
	b.mu.Unlock()
	go func() {
		b.queryAllDeviceStates()
		b.save()
	}()
	return nil
}

// Disconnect marks the backend disconnected.
func (b *Backend) Disconnect() {
	b.mu.Lock()
	b.connected = false
	b.mu.Unlock()
	b.save()
}

// IsConnected reports whether the backend is connected.
func (b *Backend) IsConnected() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.connected
}

// NumSwitches returns the number of Mi devices.
func (b *Backend) NumSwitches() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.devices)
}

// GetName returns the device name.
func (b *Backend) GetName(id int) string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if id < 0 || id >= len(b.devices) {
		return ""
	}
	return b.devices[id].Name
}

// SetName sets the name for device id.
func (b *Backend) SetName(id int, name string) error {
	if id < 0 || id >= len(b.devices) {
		return fmt.Errorf("invalid device id %d", id)
	}
	b.mu.Lock()
	b.devices[id].Name = name
	b.mu.Unlock()
	b.save()
	return nil
}

// GetDescription returns the description for device id.
// Falls back to the device name if no description is set.
func (b *Backend) GetDescription(id int) string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if id < 0 || id >= len(b.devices) {
		return ""
	}
	if b.devices[id].Description != "" {
		return b.devices[id].Description
	}
	return b.devices[id].Name
}

// GetCanWrite reports whether device id is writable.
func (b *Backend) GetCanWrite(id int) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if id < 0 || id >= len(b.devices) {
		return false
	}
	return b.devices[id].Canwrite
}

// GetMin returns the minimum value for device id.
func (b *Backend) GetMin(id int) float64 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if id < 0 || id >= len(b.devices) {
		return 0
	}
	return float64(b.devices[id].Min)
}

// GetMax returns the maximum value for device id.
func (b *Backend) GetMax(id int) float64 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if id < 0 || id >= len(b.devices) {
		return 1
	}
	return float64(b.devices[id].Max)
}

// GetStep returns the step size for device id.
func (b *Backend) GetStep(id int) float64 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if id < 0 || id >= len(b.devices) {
		return 1
	}
	return float64(b.devices[id].Step)
}

// GetSwitch returns the on/off state of device id.
func (b *Backend) GetSwitch(id int) (bool, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if id < 0 || id >= len(b.devices) {
		return false, fmt.Errorf("invalid device id %d", id)
	}
	if b.devices[id].Max > 1 {
		return false, errors.New("device is not a simple on/off switch")
	}
	return b.devices[id].Value != 0, nil
}

// GetSwitchValue returns the numeric value of device id.
func (b *Backend) GetSwitchValue(id int) (float64, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if id < 0 || id >= len(b.devices) {
		return 0, fmt.Errorf("invalid device id %d", id)
	}
	return float64(b.devices[id].Value), nil
}

// SetSwitch turns device id on or off.
// A per-device lock ensures rapid successive calls are serialised so that
// the second command always uses a fresh discovery stamp from the device.
func (b *Backend) SetSwitch(id int, state bool) error {
	if id < 0 || id >= len(b.devices) {
		return fmt.Errorf("invalid device id %d", id)
	}
	b.deviceLock[id].Lock()
	defer b.deviceLock[id].Unlock()

	b.mu.RLock()
	devices := make([]Device, len(b.devices))
	copy(devices, b.devices)
	b.mu.RUnlock()
	if err := miOnOff(int32(id), devices, state); err != nil {
		return err
	}
	b.mu.Lock()
	if state {
		b.devices[id].Value = 1
	} else {
		b.devices[id].Value = 0
	}
	b.mu.Unlock()
	b.save()
	log.Printf("[mi] switch %d set to %v", id, state)
	return nil
}

// SetSwitchValue sets device id by numeric value (0 = off, non-zero = on).
func (b *Backend) SetSwitchValue(id int, value float64) error {
	return b.SetSwitch(id, value != 0)
}

// Devices returns a copy of the device list (for config serialisation).
func (b *Backend) Devices() []Device {
	b.mu.RLock()
	defer b.mu.RUnlock()
	cp := make([]Device, len(b.devices))
	copy(cp, b.devices)
	return cp
}

// queryAllDeviceStates fetches live power state from all Mi devices in parallel.
func (b *Backend) queryAllDeviceStates() {
	log.Println("[mi] querying device states...")
	b.mu.RLock()
	devices := make([]Device, len(b.devices))
	copy(devices, b.devices)
	b.mu.RUnlock()

	var wg sync.WaitGroup
	for i := range devices {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			state, err := miQueryPower(int32(i), devices)
			if err != nil {
				log.Printf("[mi] warning: device %d query failed: %v (keeping cached value)", i, err)
				return
			}
			b.mu.Lock()
			if state {
				b.devices[i].Value = 1
			} else {
				b.devices[i].Value = 0
			}
			name := b.devices[i].Name
			b.mu.Unlock()
			log.Printf("[mi] device %d (%s): %v", i, name, state)
		}(i)
	}
	wg.Wait()
	log.Println("[mi] device state query complete")
}

// save persists device state to savePath (if set).
func (b *Backend) save() {
	if b.savePath == "" {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	data, err := json.MarshalIndent(b.devices, "", "    ")
	if err != nil {
		log.Printf("[mi] save error: %v", err)
		return
	}
	if err := os.WriteFile(b.savePath, data, 0644); err != nil {
		log.Printf("[mi] save error: %v", err)
	}
}
