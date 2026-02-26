package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"

	"alpaca-switch/backend"

	"github.com/julienschmidt/httprouter"
)

// Server is the ASCOM Alpaca HTTP API server.
type Server struct {
	router              *backend.Router
	serverTransactionID uint32
}

// New creates a Server backed by the given backend Router.
func New(r *backend.Router) *Server {
	return &Server{router: r}
}

// Start registers all routes and begins listening on addr (e.g. ":11111").
func (s *Server) Start(addr string) {
	r := httprouter.New()
	s.configureManagementAPI(r)
	s.configureCommonAPI(r)
	s.configureSwitchAPI(r)
	log.Printf("Alpaca API server listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}

func (s *Server) nextTxnID() uint32 {
	return atomic.AddUint32(&s.serverTransactionID, 1)
}

func (s *Server) prepareResponse(r *http.Request, resp *alpacaResponse) {
	resp.ClientTransactionID = uint32(getClientTransactionID(r))
	resp.ServerTransactionID = s.nextTxnID()
}

func (s *Server) sendJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// ---------- request helpers ----------

func getClientID(r *http.Request) int {
	v := getParamAnyCase(r, "ClientID")
	if v == "" {
		return -1
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return -1
	}
	return n
}

func getClientTransactionID(r *http.Request) int {
	v := getParamAnyCase(r, "ClientTransactionID")
	if v == "" {
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

func getSwitchID(r *http.Request) (int, error) {
	v := getParamAnyCase(r, "Id")
	if v == "" {
		return -1, errors.New("Id parameter missing")
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return -1, fmt.Errorf("Id parameter invalid: %s", v)
	}
	return n, nil
}

func getSwitchState(r *http.Request) (bool, error) {
	v := getParamAnyCase(r, "State")
	if v == "" {
		return false, errors.New("State parameter missing")
	}
	return strconv.ParseBool(v)
}

func getSwitchName(r *http.Request) (string, error) {
	v := getParamAnyCase(r, "Name")
	if v == "" {
		return "", errors.New("Name parameter missing")
	}
	return v, nil
}

func getSwitchValue(r *http.Request) (float64, error) {
	v := getParamAnyCase(r, "Value")
	if v == "" {
		return 0, errors.New("Value parameter missing")
	}
	return strconv.ParseFloat(v, 64)
}

func getConnected(r *http.Request) (bool, error) {
	v := getParamAnyCase(r, "Connected")
	if v == "" {
		return false, errors.New("Connected parameter missing")
	}
	return strconv.ParseBool(v)
}

func getParamAnyCase(r *http.Request, name string) string {
	if r.Method == http.MethodGet {
		return getQueryAnyCase(r, name)
	}
	return getFormAnyCase(r, name)
}

func getQueryAnyCase(r *http.Request, name string) string {
	q := r.URL.Query()
	if v := q.Get(name); v != "" {
		return v
	}
	for k, values := range q {
		if strings.EqualFold(k, name) && len(values) > 0 {
			return values[0]
		}
	}
	return ""
}

func getFormAnyCase(r *http.Request, name string) string {
	if err := r.ParseForm(); err != nil {
		return ""
	}
	if v := r.Form.Get(name); v != "" {
		return v
	}
	for k, values := range r.Form {
		if strings.EqualFold(k, name) && len(values) > 0 {
			return values[0]
		}
	}
	return ""
}

// handleNotSupported returns 400 for unsupported ASCOM actions.
func (s *Server) handleNotSupported(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var resp stringResponse
	s.prepareResponse(r, &resp.alpacaResponse)
	resp.Value = "not supported"
	resp.ErrorNumber = 0x400
	resp.ErrorMessage = "action not supported"
	s.sendJSON(w, http.StatusBadRequest, resp)
}
