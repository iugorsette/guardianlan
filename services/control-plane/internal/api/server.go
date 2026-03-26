package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"

	"github.com/sette/guardian-lan/services/control-plane/internal/domain"
	"github.com/sette/guardian-lan/services/control-plane/internal/repository"
	"github.com/sette/guardian-lan/services/control-plane/internal/service"
)

type Server struct {
	httpServer   *http.Server
	store        repository.Store
	orchestrator *service.Orchestrator
}

func NewServer(addr string, store repository.Store, orchestrator *service.Orchestrator) *Server {
	server := &Server{
		store:        store,
		orchestrator: orchestrator,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", server.handleHealthz)
	mux.HandleFunc("GET /devices", server.handleListDevices)
	mux.HandleFunc("GET /devices/{id}", server.handleGetDevice)
	mux.HandleFunc("POST /devices/{id}/profile", server.handleUpdateDeviceProfile)
	mux.HandleFunc("GET /activity/dns", server.handleListDNSEvents)
	mux.HandleFunc("GET /activity/flows", server.handleListFlowEvents)
	mux.HandleFunc("GET /alerts", server.handleListAlerts)
	mux.HandleFunc("POST /alerts/{id}/ack", server.handleAckAlert)

	server.httpServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return server
}

func (s *Server) ListenAndServe() error {
	err := s.httpServer.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleListDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := s.store.ListDevices(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, devices)
}

func (s *Server) handleGetDevice(w http.ResponseWriter, r *http.Request) {
	device, err := s.store.GetDevice(r.Context(), r.PathValue("id"))
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, device)
}

func (s *Server) handleUpdateDeviceProfile(w http.ResponseWriter, r *http.Request) {
	var request domain.ProfileUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if request.ProfileID == "" {
		writeErrorMessage(w, http.StatusBadRequest, "profile_id is required")
		return
	}

	device, err := s.orchestrator.UpdateDeviceProfile(r.Context(), r.PathValue("id"), request.ProfileID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, device)
}

func (s *Server) handleListDNSEvents(w http.ResponseWriter, r *http.Request) {
	events, err := s.store.ListDNSEvents(r.Context(), parseLimit(r, 50))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, events)
}

func (s *Server) handleListFlowEvents(w http.ResponseWriter, r *http.Request) {
	events, err := s.store.ListFlowEvents(r.Context(), parseLimit(r, 50))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, events)
}

func (s *Server) handleListAlerts(w http.ResponseWriter, r *http.Request) {
	alerts, err := s.store.ListAlerts(r.Context(), parseLimit(r, 50), r.URL.Query().Get("status"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, alerts)
}

func (s *Server) handleAckAlert(w http.ResponseWriter, r *http.Request) {
	alert, err := s.store.AckAlert(r.Context(), r.PathValue("id"))
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, alert)
}

func parseLimit(r *http.Request, fallback int) int {
	value := r.URL.Query().Get("limit")
	if value == "" {
		return fallback
	}

	limit, err := strconv.Atoi(value)
	if err != nil || limit <= 0 {
		return fallback
	}

	return limit
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeErrorMessage(w, status, err.Error())
}

func writeErrorMessage(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
