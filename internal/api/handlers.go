package api

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/supminer/suptunnels/internal/config"
	"github.com/supminer/suptunnels/internal/metrics"
)

type API struct {
	cfg       *config.Config
	collector *metrics.Collector
	mu        sync.Mutex
}

func NewAPI(cfg *config.Config, collector *metrics.Collector) *API {
	return &API{
		cfg:       cfg,
		collector: collector,
	}
}

func (a *API) Handlers() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/tunnels", a.handleTunnels)
	mux.HandleFunc("/api/tunnels/", a.handleTunnelByID)
	mux.HandleFunc("/api/stats", a.handleStats)
	mux.HandleFunc("/api/ws", a.handleWS)
	return a.authMiddleware(mux)
}

func (a *API) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-API-Token")
		if token != a.cfg.Server.Secret {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *API) handleTunnels(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		json.NewEncoder(w).Encode(a.cfg.Tunnels)
	case http.MethodPost:
		var t config.TunnelConfig
		if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		t.ID = uuid.New().String()
		a.mu.Lock()
		a.cfg.Tunnels = append(a.cfg.Tunnels, t)
		a.mu.Unlock()
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(t)
	}
}

func (a *API) handleTunnelByID(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/api/tunnels/"):]
	switch r.Method {
	case http.MethodDelete:
		a.mu.Lock()
		for i, t := range a.cfg.Tunnels {
			if t.ID == id {
				a.cfg.Tunnels = append(a.cfg.Tunnels[:i], a.cfg.Tunnels[i+1:]...)
				break
			}
		}
		a.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}
}

func (a *API) handleStats(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(a.collector.GetStats())
}
