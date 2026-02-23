package server

import (
	"fmt"
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (s *Server) configureSwitchAPI(r *httprouter.Router) {
	r.GET("/setup/v1/switch/0/setup", s.handleSetup)
	r.GET("/api/v1/switch/0/maxswitch", s.handleMaxSwitch)
	r.GET("/api/v1/switch/0/canwrite", s.handleCanWrite)
	r.GET("/api/v1/switch/0/getswitch", s.handleGetSwitch)
	r.GET("/api/v1/switch/0/getswitchdescription", s.handleGetSwitchDescription)
	r.GET("/api/v1/switch/0/getswitchname", s.handleGetSwitchName)
	r.GET("/api/v1/switch/0/getswitchvalue", s.handleGetSwitchValue)
	r.GET("/api/v1/switch/0/minswitchvalue", s.handleMinSwitchValue)
	r.GET("/api/v1/switch/0/maxswitchvalue", s.handleMaxSwitchValue)
	r.GET("/api/v1/switch/0/switchstep", s.handleSwitchStep)
	r.PUT("/api/v1/switch/0/setswitch", s.handleSetSwitch)
	r.PUT("/api/v1/switch/0/setswitchname", s.handleSetSwitchName)
	r.PUT("/api/v1/switch/0/setswitchvalue", s.handleSetSwitchValue)
}

func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprintln(w, serverName)
}

func (s *Server) handleMaxSwitch(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	resp := int32Response{Value: int32(s.router.NumSwitches())}
	s.prepareResponse(r, &resp.alpacaResponse)
	s.sendJSON(w, http.StatusOK, resp)
}

func (s *Server) handleCanWrite(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	id, err := getSwitchID(r)
	if err != nil {
		s.badRequest(w, r, err)
		return
	}
	resp := booleanResponse{Value: s.router.GetCanWrite(id)}
	s.prepareResponse(r, &resp.alpacaResponse)
	s.sendJSON(w, http.StatusOK, resp)
}

func (s *Server) handleGetSwitch(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	id, err := getSwitchID(r)
	if err != nil {
		s.badRequest(w, r, err)
		return
	}
	state, err := s.router.GetSwitch(id)
	if err != nil {
		s.badRequest(w, r, err)
		return
	}
	resp := booleanResponse{Value: state}
	s.prepareResponse(r, &resp.alpacaResponse)
	s.sendJSON(w, http.StatusOK, resp)
}

func (s *Server) handleGetSwitchDescription(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	id, err := getSwitchID(r)
	if err != nil {
		s.badRequest(w, r, err)
		return
	}
	resp := stringResponse{Value: s.router.GetDescription(id)}
	s.prepareResponse(r, &resp.alpacaResponse)
	s.sendJSON(w, http.StatusOK, resp)
}

func (s *Server) handleGetSwitchName(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	id, err := getSwitchID(r)
	if err != nil {
		s.badRequest(w, r, err)
		return
	}
	resp := stringResponse{Value: s.router.GetName(id)}
	s.prepareResponse(r, &resp.alpacaResponse)
	s.sendJSON(w, http.StatusOK, resp)
}

func (s *Server) handleGetSwitchValue(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	id, err := getSwitchID(r)
	if err != nil {
		s.badRequest(w, r, err)
		return
	}
	val, err := s.router.GetSwitchValue(id)
	if err != nil {
		s.badRequest(w, r, err)
		return
	}
	resp := doubleResponse{Value: val}
	s.prepareResponse(r, &resp.alpacaResponse)
	s.sendJSON(w, http.StatusOK, resp)
}

func (s *Server) handleMinSwitchValue(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	id, err := getSwitchID(r)
	if err != nil {
		s.badRequest(w, r, err)
		return
	}
	resp := doubleResponse{Value: s.router.GetMin(id)}
	s.prepareResponse(r, &resp.alpacaResponse)
	s.sendJSON(w, http.StatusOK, resp)
}

func (s *Server) handleMaxSwitchValue(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	id, err := getSwitchID(r)
	if err != nil {
		s.badRequest(w, r, err)
		return
	}
	resp := doubleResponse{Value: s.router.GetMax(id)}
	s.prepareResponse(r, &resp.alpacaResponse)
	s.sendJSON(w, http.StatusOK, resp)
}

func (s *Server) handleSwitchStep(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	id, err := getSwitchID(r)
	if err != nil {
		s.badRequest(w, r, err)
		return
	}
	resp := doubleResponse{Value: s.router.GetStep(id)}
	s.prepareResponse(r, &resp.alpacaResponse)
	s.sendJSON(w, http.StatusOK, resp)
}

func (s *Server) handleSetSwitch(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	log.Print("[server] SetSwitch called")
	id, err := getSwitchID(r)
	if err != nil {
		s.badRequest(w, r, err)
		return
	}
	state, err := getSwitchState(r)
	if err != nil {
		s.badRequest(w, r, err)
		return
	}
	log.Printf("[server] SetSwitch id=%d state=%v", id, state)
	if err := s.router.SetSwitch(id, state); err != nil {
		s.badRequest(w, r, err)
		return
	}
	var resp putResponse
	s.prepareResponse(r, &resp.alpacaResponse)
	s.sendJSON(w, http.StatusOK, resp)
}

func (s *Server) handleSetSwitchName(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	id, err := getSwitchID(r)
	if err != nil {
		s.badRequest(w, r, err)
		return
	}
	name, err := getSwitchName(r)
	if err != nil {
		s.badRequest(w, r, err)
		return
	}
	if err := s.router.SetName(id, name); err != nil {
		s.badRequest(w, r, err)
		return
	}
	var resp putResponse
	s.prepareResponse(r, &resp.alpacaResponse)
	s.sendJSON(w, http.StatusOK, resp)
}

func (s *Server) handleSetSwitchValue(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	id, err := getSwitchID(r)
	if err != nil {
		s.badRequest(w, r, err)
		return
	}
	val, err := getSwitchValue(r)
	if err != nil {
		s.badRequest(w, r, err)
		return
	}
	if err := s.router.SetSwitchValue(id, val); err != nil {
		s.badRequest(w, r, err)
		return
	}
	var resp putResponse
	s.prepareResponse(r, &resp.alpacaResponse)
	s.sendJSON(w, http.StatusOK, resp)
}

// badRequest sends a 400 response with the error message.
func (s *Server) badRequest(w http.ResponseWriter, r *http.Request, err error) {
	resp := stringResponse{Value: err.Error()}
	s.prepareResponse(r, &resp.alpacaResponse)
	resp.ErrorNumber = 0x400
	resp.ErrorMessage = err.Error()
	s.sendJSON(w, http.StatusBadRequest, resp)
}
