// Package api provides HTTP API handlers.
package api

import (
	"encoding/json"
	"net/http"
	"time"

	"timesphere/internal/ntp"
)

// TimeResponse contains time information.
type TimeResponse struct {
	UTC         string `json:"utc"`
	ISO8601     string `json:"iso8601"`
	Epoch       int64  `json:"epoch"`
	Nanoseconds int64  `json:"nanoseconds"`
}

// StatusResponse contains application status.
type StatusResponse struct {
	Status      string    `json:"status"`
	Version     string    `json:"version"`
	Uptime      string    `json:"uptime"`
	Servers     int       `json:"servers"`
	Reachable   int       `json:"reachable"`
	Timestamp   time.Time `json:"timestamp"`
}

// ServerResponse contains server status information.
type ServerResponse struct {
	Name           string        `json:"name"`
	Address        string        `json:"address"`
	Offset         int64         `json:"offset"`          // Nanoseconds
	Delay          int64         `json:"delay"`           // Nanoseconds
	Jitter         int64         `json:"jitter"`          // Nanoseconds
	Reachable      bool          `json:"reachable"`       // Server is reachable
	PollInterval   int           `json:"poll_interval"`   // Poll interval in seconds
	Stratum        uint8         `json:"stratum"`         // Server stratum
	LeapIndicator  uint8         `json:"leap_indicator"`  // Leap indicator
	RootDispersion int64         `json:"root_dispersion"` // Nanoseconds
	Precision      int64         `json:"precision"`       // Nanoseconds
	LastQuery      time.Time     `json:"last_query"`      // Last successful query
	Error          string        `json:"error,omitempty"` // Error message if any
	RTT            int64         `json:"rtt"`             // Round trip time nanoseconds
}

// Handler holds API dependencies.
type Handler struct {
	monitor *ntp.Monitor
	startTime time.Time
	version   string
}

// NewHandler creates a new API handler.
func NewHandler(monitor *ntp.Monitor, version string) *Handler {
	return &Handler{
		monitor:   monitor,
		startTime: time.Now(),
		version:   version,
	}
}

// GetTime returns current time information.
func (h *Handler) GetTime(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()
	resp := TimeResponse{
		UTC:         now.Format(time.RFC3339),
		ISO8601:     now.Format("2006-01-02T15:04:05Z07:00"),
		Epoch:       now.Unix(),
		Nanoseconds: now.UnixNano(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GetStatus returns application status.
func (h *Handler) GetStatus(w http.ResponseWriter, r *http.Request) {
	statuses := h.monitor.GetStatuses()
	reachable := 0
	for _, s := range statuses {
		if s.Reachable {
			reachable++
		}
	}

	resp := StatusResponse{
		Status:    "ok",
		Version:   h.version,
		Uptime:    time.Since(h.startTime).String(),
		Servers:   len(statuses),
		Reachable: reachable,
		Timestamp: time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GetServers returns all monitored servers.
func (h *Handler) GetServers(w http.ResponseWriter, r *http.Request) {
	statuses := h.monitor.GetStatuses()
	servers := make([]ServerResponse, 0, len(statuses))

	for _, s := range statuses {
		srv := ServerResponse{
			Name:           s.Name,
			Address:        s.Address,
			Offset:         s.Offset.Nanoseconds(),
			Delay:          s.Delay.Nanoseconds(),
			Jitter:         s.Jitter.Nanoseconds(),
			Reachable:      s.Reachable,
			PollInterval:   s.PollInterval,
			Stratum:        s.Stratum,
			LeapIndicator:  s.LeapIndicator,
			RootDispersion: s.RootDispersion.Nanoseconds(),
			Precision:      s.Precision.Nanoseconds(),
			LastQuery:      s.LastQuery,
			Error:          s.Error,
			RTT:            s.RTT.Nanoseconds(),
		}
		servers = append(servers, srv)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(servers)
}

// HealthCheck returns health status.
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

// ReadinessCheck returns readiness status.
func (h *Handler) ReadinessCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"ready": "true"})
}
