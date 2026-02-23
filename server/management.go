package server

import (
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

const (
	serverName     = "Alpaca Switch Controller"
	manufacturer   = "https://github.com/exploded/"
	driverVersion  = "1.0.0"
	location       = "Earth"
	deviceUniqueID = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
)

func (s *Server) configureManagementAPI(r *httprouter.Router) {
	r.GET("/", s.handleRoot)
	r.GET("/management/apiversions", s.handleAPIVersions)
	r.GET("/management/v1/description", s.handleDescription)
	r.GET("/management/v1/configureddevices", s.handleConfiguredDevices)
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprintln(w, serverName)
}

func (s *Server) handleAPIVersions(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	resp := uint32ListResponse{Value: []uint32{1}}
	s.prepareResponse(r, &resp.alpacaResponse)
	s.sendJSON(w, http.StatusOK, resp)
}

func (s *Server) handleDescription(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	resp := managementDescriptionResponse{
		Value: ServerDescription{
			ServerName:          serverName,
			Manufacturer:        manufacturer,
			ManufacturerVersion: driverVersion,
			Location:            location,
		},
	}
	s.prepareResponse(r, &resp.alpacaResponse)
	s.sendJSON(w, http.StatusOK, resp)
}

func (s *Server) handleConfiguredDevices(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	resp := managementDevicesListResponse{
		Value: []DeviceConfiguration{
			{
				DeviceName:   serverName,
				DeviceType:   "Switch",
				DeviceNumber: 0,
				UniqueID:     deviceUniqueID,
			},
		},
	}
	s.prepareResponse(r, &resp.alpacaResponse)
	s.sendJSON(w, http.StatusOK, resp)
}
