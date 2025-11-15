package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// APIServer provides HTTP API for daemon control
type APIServer struct {
	addr    string
	daemon  *Daemon
	logger  *zap.Logger
	server  *http.Server
	handler *APIHandler
}

// APIHandler handles API requests
type APIHandler struct {
	daemon *Daemon
	logger *zap.Logger
}

// NewAPIServer creates a new API server
func NewAPIServer(addr string, daemon *Daemon, logger *zap.Logger) *APIServer {
	handler := &APIHandler{
		daemon: daemon,
		logger: logger,
	}

	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", handler.handleHealth)

	// Status
	mux.HandleFunc("/api/v1/status", handler.handleStatus)

	// Chart substitutions
	mux.HandleFunc("/api/v1/charts", handler.handleCharts)
	mux.HandleFunc("/api/v1/charts/remove", handler.handleRemoveChart)

	// Image substitutions
	mux.HandleFunc("/api/v1/images", handler.handleImages)
	mux.HandleFunc("/api/v1/images/remove", handler.handleRemoveImage)

	// Substitutions list
	mux.HandleFunc("/api/v1/substitutions", handler.handleSubstitutions)

	// Sync
	mux.HandleFunc("/api/v1/sync", handler.handleSync)

	// Drift reports
	mux.HandleFunc("/api/v1/drift", handler.handleDrift)

	// Reload
	mux.HandleFunc("/api/v1/reload", handler.handleReload)

	// Shutdown
	mux.HandleFunc("/api/v1/shutdown", handler.handleShutdown)

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return &APIServer{
		addr:    addr,
		daemon:  daemon,
		logger:  logger,
		server:  server,
		handler: handler,
	}
}

// Start starts the API server
func (s *APIServer) Start() error {
	go func() {
		s.logger.Info("API server listening", zap.String("addr", s.addr))
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("API server error", zap.Error(err))
		}
	}()
	return nil
}

// Stop stops the API server
func (s *APIServer) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}

// handleHealth handles health check requests
func (h *APIHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

// handleStatus handles status requests
func (h *APIHandler) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := h.daemon.GetStatus()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handleCharts handles chart substitution requests
func (h *APIHandler) handleCharts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req AddChartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	substitutor := h.daemon.GetSubstitutor()
	if err := substitutor.AddChartSubstitution(req.Original, req.LocalPath); err != nil {
		h.sendError(w, fmt.Sprintf("Failed to add chart substitution: %v", err), http.StatusBadRequest)
		return
	}

	h.logger.Info("chart substitution added via API",
		zap.String("original", req.Original),
		zap.String("local", req.LocalPath))

	h.sendSuccess(w, fmt.Sprintf("Chart substitution added: %s → %s", req.Original, req.LocalPath))
}

// handleRemoveChart handles chart substitution removal
func (h *APIHandler) handleRemoveChart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RemoveChartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	substitutor := h.daemon.GetSubstitutor()
	if err := substitutor.RemoveChartSubstitution(req.Original); err != nil {
		h.sendError(w, fmt.Sprintf("Failed to remove chart substitution: %v", err), http.StatusBadRequest)
		return
	}

	h.logger.Info("chart substitution removed via API", zap.String("original", req.Original))
	h.sendSuccess(w, fmt.Sprintf("Chart substitution removed: %s", req.Original))
}

// handleImages handles image substitution requests
func (h *APIHandler) handleImages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req AddImageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	substitutor := h.daemon.GetSubstitutor()
	if err := substitutor.AddImageSubstitution(req.Original, req.Replacement); err != nil {
		h.sendError(w, fmt.Sprintf("Failed to add image substitution: %v", err), http.StatusBadRequest)
		return
	}

	h.logger.Info("image substitution added via API",
		zap.String("original", req.Original),
		zap.String("replacement", req.Replacement))

	h.sendSuccess(w, fmt.Sprintf("Image substitution added: %s → %s", req.Original, req.Replacement))
}

// handleRemoveImage handles image substitution removal
func (h *APIHandler) handleRemoveImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RemoveImageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	substitutor := h.daemon.GetSubstitutor()
	if err := substitutor.RemoveImageSubstitution(req.Original); err != nil {
		h.sendError(w, fmt.Sprintf("Failed to remove image substitution: %v", err), http.StatusBadRequest)
		return
	}

	h.logger.Info("image substitution removed via API", zap.String("original", req.Original))
	h.sendSuccess(w, fmt.Sprintf("Image substitution removed: %s", req.Original))
}

// handleSubstitutions handles listing all substitutions
func (h *APIHandler) handleSubstitutions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	substitutor := h.daemon.GetSubstitutor()

	charts := substitutor.ListChartSubstitutions()
	images := substitutor.ListImageSubstitutions()

	response := SubstitutionsResponse{
		Charts: make([]ChartSubstitution, len(charts)),
		Images: make([]ImageSubstitution, len(images)),
	}

	for i, c := range charts {
		response.Charts[i] = ChartSubstitution{
			Original:  c.Original,
			LocalPath: c.LocalPath,
		}
	}

	for i, img := range images {
		response.Images[i] = ImageSubstitution{
			Original:    img.Original,
			Replacement: img.Replacement,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleSync handles manual sync requests
func (h *APIHandler) handleSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// TODO: Implement sync functionality
	// This would require access to the sync executor
	h.logger.Info("sync requested via API", zap.Bool("dryRun", req.DryRun))
	h.sendSuccess(w, "Sync functionality not yet implemented in daemon mode")
}

// handleDrift handles drift report requests
func (h *APIHandler) handleDrift(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	detector := h.daemon.GetDetector()
	if detector == nil {
		h.sendError(w, "Drift detection not enabled", http.StatusBadRequest)
		return
	}

	// TODO: Implement drift report retrieval
	// This would require storing drift reports in the detector
	h.sendSuccess(w, "Drift report retrieval not yet implemented")
}

// handleReload handles helmfile reload requests
func (h *APIHandler) handleReload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	manager := h.daemon.GetManager()
	if err := manager.Load(); err != nil {
		h.sendError(w, fmt.Sprintf("Failed to reload helmfile: %v", err), http.StatusInternalServerError)
		return
	}

	h.logger.Info("helmfile reloaded via API")
	h.sendSuccess(w, "Helmfile reloaded successfully")
}

// handleShutdown handles graceful shutdown requests
func (h *APIHandler) handleShutdown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.logger.Info("shutdown requested via API")
	h.sendSuccess(w, "Shutting down...")

	// Trigger shutdown in a goroutine so we can respond first
	go func() {
		time.Sleep(100 * time.Millisecond)
		h.daemon.shutdownCh <- nil
	}()
}

// sendError sends an error response
func (h *APIHandler) sendError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}

// sendSuccess sends a success response
func (h *APIHandler) sendSuccess(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(SuccessResponse{Message: message})
}
