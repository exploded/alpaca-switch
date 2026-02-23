package server

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (s *Server) configureCommonAPI(r *httprouter.Router) {
	// Unsupported ASCOM common actions
	r.PUT("/api/v1/switch/0/action", s.handleNotSupported)
	r.PUT("/api/v1/switch/0/commandblind", s.handleNotSupported)
	r.PUT("/api/v1/switch/0/commandbool", s.handleNotSupported)
	r.PUT("/api/v1/switch/0/commandstring", s.handleNotSupported)

	// Connection
	r.GET("/api/v1/switch/0/connected", s.handleGetConnected)
	r.PUT("/api/v1/switch/0/connected", s.handleSetConnected)

	// Device info
	r.GET("/api/v1/switch/0/description", s.handleDeviceDescription)
	r.GET("/api/v1/switch/0/driverinfo", s.handleDriverInfo)
	r.GET("/api/v1/switch/0/driverversion", s.handleDriverVersion)
	r.GET("/api/v1/switch/0/interfaceversion", s.handleInterfaceVersion)
	r.GET("/api/v1/switch/0/name", s.handleName)
	r.GET("/api/v1/switch/0/supportedactions", s.handleSupportedActions)
}

func (s *Server) handleGetConnected(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Report connected if ALL backends are connected
	connected := true
	for _, b := range s.router.Backends() {
		if !b.IsConnected() {
			connected = false
			break
		}
	}
	resp := booleanResponse{Value: connected}
	s.prepareResponse(r, &resp.alpacaResponse)
	s.sendJSON(w, http.StatusOK, resp)
}

func (s *Server) handleSetConnected(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	connect, err := getConnected(r)
	if err != nil {
		resp := stringResponse{Value: err.Error()}
		s.prepareResponse(r, &resp.alpacaResponse)
		resp.ErrorNumber = 0x400
		resp.ErrorMessage = err.Error()
		s.sendJSON(w, http.StatusBadRequest, resp)
		return
	}
	for _, b := range s.router.Backends() {
		if connect {
			_ = b.Connect()
		} else {
			b.Disconnect()
		}
	}
	var resp putResponse
	s.prepareResponse(r, &resp.alpacaResponse)
	s.sendJSON(w, http.StatusOK, resp)
}

func (s *Server) handleDeviceDescription(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	resp := stringResponse{Value: serverName + " — controls Xiaomi Mi smart plugs and Hikvision IR cameras via ASCOM Alpaca"}
	s.prepareResponse(r, &resp.alpacaResponse)
	s.sendJSON(w, http.StatusOK, resp)
}

func (s *Server) handleDriverInfo(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	resp := stringResponse{Value: serverName + " v" + driverVersion + " — " + manufacturer}
	s.prepareResponse(r, &resp.alpacaResponse)
	s.sendJSON(w, http.StatusOK, resp)
}

func (s *Server) handleDriverVersion(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	resp := stringResponse{Value: driverVersion}
	s.prepareResponse(r, &resp.alpacaResponse)
	s.sendJSON(w, http.StatusOK, resp)
}

func (s *Server) handleInterfaceVersion(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	resp := int32Response{Value: 2} // ISwitchV2
	s.prepareResponse(r, &resp.alpacaResponse)
	s.sendJSON(w, http.StatusOK, resp)
}

func (s *Server) handleName(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	resp := stringResponse{Value: serverName}
	s.prepareResponse(r, &resp.alpacaResponse)
	s.sendJSON(w, http.StatusOK, resp)
}

func (s *Server) handleSupportedActions(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	resp := stringListResponse{Value: []string{}}
	s.prepareResponse(r, &resp.alpacaResponse)
	s.sendJSON(w, http.StatusOK, resp)
}
