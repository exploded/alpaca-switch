package backend

import "fmt"

// SwitchBackend is the interface all hardware backends must implement.
// Each backend manages one or more named switches (0-based local IDs).
type SwitchBackend interface {
	// NumSwitches returns the total number of switches this backend controls.
	NumSwitches() int

	// GetName returns the display name for switch id.
	GetName(id int) string

	// SetName sets a custom name for switch id.
	SetName(id int, name string) error

	// GetDescription returns a description for switch id.
	GetDescription(id int) string

	// GetCanWrite reports whether switch id can be written.
	GetCanWrite(id int) bool

	// GetMin returns the minimum value for switch id.
	GetMin(id int) float64

	// GetMax returns the maximum value for switch id.
	GetMax(id int) float64

	// GetStep returns the step size for switch id.
	GetStep(id int) float64

	// GetSwitch returns the boolean on/off state of switch id.
	GetSwitch(id int) (bool, error)

	// GetSwitchValue returns the numeric value of switch id.
	GetSwitchValue(id int) (float64, error)

	// SetSwitch sets the on/off state of switch id.
	SetSwitch(id int, state bool) error

	// SetSwitchValue sets the numeric value of switch id.
	SetSwitchValue(id int, value float64) error

	// Connect initialises the backend and connects to hardware.
	Connect() error

	// Disconnect tears down the hardware connection.
	Disconnect()

	// IsConnected reports whether the backend is currently connected.
	IsConnected() bool
}

// Router maps flat global switch IDs to the correct backend and local ID.
type Router struct {
	backends []SwitchBackend
	// index[globalID] = {backendIdx, localID}
	index []switchRef
}

type switchRef struct {
	backend  SwitchBackend
	localID  int
}

// NewRouter builds a Router from an ordered list of backends.
func NewRouter(backends []SwitchBackend) *Router {
	r := &Router{backends: backends}
	for _, b := range backends {
		for localID := 0; localID < b.NumSwitches(); localID++ {
			r.index = append(r.index, switchRef{backend: b, localID: localID})
		}
	}
	return r
}

// NumSwitches returns the total number of switches across all backends.
func (r *Router) NumSwitches() int { return len(r.index) }

// Backends returns all registered backends.
func (r *Router) Backends() []SwitchBackend { return r.backends }

func (r *Router) ref(globalID int) (switchRef, bool) {
	if globalID < 0 || globalID >= len(r.index) {
		return switchRef{}, false
	}
	return r.index[globalID], true
}

func (r *Router) GetName(id int) string {
	if ref, ok := r.ref(id); ok {
		return ref.backend.GetName(ref.localID)
	}
	return ""
}

func (r *Router) SetName(id int, name string) error {
	if ref, ok := r.ref(id); ok {
		return ref.backend.SetName(ref.localID, name)
	}
	return errInvalidID(id)
}

func (r *Router) GetDescription(id int) string {
	if ref, ok := r.ref(id); ok {
		return ref.backend.GetDescription(ref.localID)
	}
	return ""
}

func (r *Router) GetCanWrite(id int) bool {
	if ref, ok := r.ref(id); ok {
		return ref.backend.GetCanWrite(ref.localID)
	}
	return false
}

func (r *Router) GetMin(id int) float64 {
	if ref, ok := r.ref(id); ok {
		return ref.backend.GetMin(ref.localID)
	}
	return 0
}

func (r *Router) GetMax(id int) float64 {
	if ref, ok := r.ref(id); ok {
		return ref.backend.GetMax(ref.localID)
	}
	return 1
}

func (r *Router) GetStep(id int) float64 {
	if ref, ok := r.ref(id); ok {
		return ref.backend.GetStep(ref.localID)
	}
	return 1
}

func (r *Router) GetSwitch(id int) (bool, error) {
	if ref, ok := r.ref(id); ok {
		return ref.backend.GetSwitch(ref.localID)
	}
	return false, errInvalidID(id)
}

func (r *Router) GetSwitchValue(id int) (float64, error) {
	if ref, ok := r.ref(id); ok {
		return ref.backend.GetSwitchValue(ref.localID)
	}
	return 0, errInvalidID(id)
}

func (r *Router) SetSwitch(id int, state bool) error {
	if ref, ok := r.ref(id); ok {
		return ref.backend.SetSwitch(ref.localID, state)
	}
	return errInvalidID(id)
}

func (r *Router) SetSwitchValue(id int, value float64) error {
	if ref, ok := r.ref(id); ok {
		return ref.backend.SetSwitchValue(ref.localID, value)
	}
	return errInvalidID(id)
}

func errInvalidID(id int) error {
	return fmt.Errorf("switch ID %d is out of range", id)
}
