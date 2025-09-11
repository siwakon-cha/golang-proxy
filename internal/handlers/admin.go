package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"rpc-proxy/internal/database"
	"rpc-proxy/internal/repository"
	"rpc-proxy/internal/repository/gorm"
)

type AdminHandler struct {
	db           *database.GormDB
	rpcRepo      repository.RPCEndpointRepository
	settingsRepo repository.SettingsRepository
	healthRepo   repository.HealthCheckRepository
}

func NewAdminHandler(db *database.GormDB) *AdminHandler {
	return &AdminHandler{
		db:           db,
		rpcRepo:      gorm.NewRPCEndpointRepository(db),
		settingsRepo: gorm.NewSettingsRepository(db),
		healthRepo:   gorm.NewHealthCheckRepository(db),
	}
}

func (h *AdminHandler) RegisterRoutes(mux *http.ServeMux) {
	// RPC Endpoints
	mux.HandleFunc("/admin/endpoints", h.handleEndpoints)
	mux.HandleFunc("/admin/endpoints/", h.handleEndpointByID)
	
	// Settings
	mux.HandleFunc("/admin/settings", h.handleSettings)
	mux.HandleFunc("/admin/settings/", h.handleSettingByKey)
	
	// Health Checks
	mux.HandleFunc("/admin/health-checks/", h.handleHealthChecks)
}

// RPC Endpoints handlers
func (h *AdminHandler) handleEndpoints(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.listEndpoints(w, r)
	case "POST":
		h.createEndpoint(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *AdminHandler) handleEndpointByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/admin/endpoints/")
	id, err := strconv.Atoi(path)
	if err != nil {
		http.Error(w, "Invalid endpoint ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "GET":
		h.getEndpoint(w, r, id)
	case "PUT":
		h.updateEndpoint(w, r, id)
	case "DELETE":
		h.deleteEndpoint(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *AdminHandler) listEndpoints(w http.ResponseWriter, r *http.Request) {
	endpoints, err := h.rpcRepo.GetAll()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get endpoints: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": endpoints,
	})
}

func (h *AdminHandler) getEndpoint(w http.ResponseWriter, r *http.Request, id int) {
	endpoint, err := h.rpcRepo.GetByID(id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get endpoint: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": endpoint,
	})
}

func (h *AdminHandler) createEndpoint(w http.ResponseWriter, r *http.Request) {
	var req repository.CreateRPCEndpointRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Set default weight if not provided
	if req.Weight == 0 {
		req.Weight = 1
	}

	endpoint, err := h.rpcRepo.Create(&req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create endpoint: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": endpoint,
	})
}

func (h *AdminHandler) updateEndpoint(w http.ResponseWriter, r *http.Request, id int) {
	var req repository.UpdateRPCEndpointRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	endpoint, err := h.rpcRepo.Update(id, &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update endpoint: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": endpoint,
	})
}

func (h *AdminHandler) deleteEndpoint(w http.ResponseWriter, r *http.Request, id int) {
	if err := h.rpcRepo.Delete(id); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete endpoint: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Settings handlers
func (h *AdminHandler) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.listSettings(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *AdminHandler) handleSettingByKey(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/admin/settings/")
	if key == "" {
		http.Error(w, "Setting key is required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "GET":
		h.getSetting(w, r, key)
	case "PUT":
		h.updateSetting(w, r, key)
	case "DELETE":
		h.deleteSetting(w, r, key)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *AdminHandler) listSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.settingsRepo.GetAll()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get settings: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": settings,
	})
}

func (h *AdminHandler) getSetting(w http.ResponseWriter, r *http.Request, key string) {
	value, err := h.settingsRepo.Get(key)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get setting: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"key":   key,
		"value": value,
	})
}

func (h *AdminHandler) updateSetting(w http.ResponseWriter, r *http.Request, key string) {
	var req struct {
		Value       string `json:"value"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := h.settingsRepo.Set(key, req.Value, req.Description); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update setting: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"key":   key,
		"value": req.Value,
	})
}

func (h *AdminHandler) deleteSetting(w http.ResponseWriter, r *http.Request, key string) {
	if err := h.settingsRepo.Delete(key); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete setting: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Health Checks handlers
func (h *AdminHandler) handleHealthChecks(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/admin/health-checks/")
	endpointID, err := strconv.Atoi(path)
	if err != nil {
		http.Error(w, "Invalid endpoint ID", http.StatusBadRequest)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 100 // default limit
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	healthChecks, err := h.healthRepo.GetByEndpointID(endpointID, limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get health checks: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": healthChecks,
	})
}