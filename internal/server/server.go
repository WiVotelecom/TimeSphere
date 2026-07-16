// Package server provides the HTTP server implementation.
package server

import (
	"context"
	"encoding/json"
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

//go:embed static/*
var staticFS embed.FS

//go:embed templates/*
var templateFS embed.FS

// Logger interface for compatibility with Go 1.19
type Logger interface {
	Info(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	Debug(msg string, keysAndValues ...interface{})
}

// Server represents the TimeSphere application server.
type Server struct {
	cfg       *Config
	httpSrv   *http.Server
	monitor   *Monitor
	db        *DB
	api       *Handler
	metrics   *Collector
	logger    Logger
	version   string
	startTime time.Time
	mu        sync.RWMutex
}

// Config holds server configuration
type Config struct {
	Server     ServerConfig
	Servers    []ServerCfg
	Thresholds Thresholds
	UI         UIConfig
	Auth       AuthConfig
	TLS        TLSConfig
	History    HistoryConfig
	Alerting   AlertingConfig
	Discovery  DiscoveryConfig
	Logging    LoggingConfig
}

// ServerConfig contains server settings.
type ServerConfig struct {
	Port            int
	Host            string
	ReadTimeout     int
	WriteTimeout    int
	ShutdownTimeout int
}

// ServerCfg represents an NTP server to monitor.
type ServerCfg struct {
	Name    string
	Address string
	Port    int
	Enabled bool
}

// Thresholds define alerting thresholds.
type Thresholds struct {
	OffsetMS     int64
	JitterMS     int64
	PollInterval int
	ReachTimeout int
}

// UIConfig contains UI-related settings.
type UIConfig struct {
	Title       string
	DarkMode    bool
	RefreshRate int
}

// AuthConfig contains authentication settings.
type AuthConfig struct {
	Enabled  bool
	Username string
	Password string
}

// TLSConfig contains TLS settings.
type TLSConfig struct {
	Enabled  bool
	CertFile string
	KeyFile  string
}

// HistoryConfig contains history retention settings.
type HistoryConfig struct {
	Enabled      bool
	SampleRate1h int
}

// AlertingConfig contains alerting settings.
type AlertingConfig struct {
	Enabled bool
	Webhook string
}

// DiscoveryConfig contains auto-discovery settings.
type DiscoveryConfig struct {
	Enabled bool
	CIDRs   []string
}

// LoggingConfig contains logging settings.
type LoggingConfig struct {
	Level  string
	Format string
}

// Monitor manages NTP server monitoring.
type Monitor struct{}

func (m *Monitor) Start()                              {}
func (m *Monitor) Stop()                               {}
func (m *Monitor) GetStatuses() map[string]*ServerStatus { return make(map[string]*ServerStatus) }

// ServerStatus represents the status of an NTP server query.
type ServerStatus struct {
	Name           string        `json:"name"`
	Address        string        `json:"address"`
	Offset         time.Duration `json:"offset"`
	Delay          time.Duration `json:"delay"`
	Jitter         time.Duration `json:"jitter"`
	Reachable      bool          `json:"reachable"`
	PollInterval   int           `json:"poll_interval"`
	Stratum        uint8         `json:"stratum"`
	LeapIndicator  uint8         `json:"leap_indicator"`
	RootDispersion time.Duration `json:"root_dispersion"`
	Precision      time.Duration `json:"precision"`
	LastQuery      time.Time     `json:"last_query"`
	Error          string        `json:"error,omitempty"`
	RTT            time.Duration `json:"rtt"`
}

// DB wraps database operations.
type DB struct{}

func (db *DB) Close() error { return nil }

// Handler holds API dependencies.
type Handler struct {
	monitor   *Monitor
	startTime time.Time
	version   string
}

func NewHandler(monitor *Monitor, version string) *Handler {
	return &Handler{
		monitor:   monitor,
		startTime: time.Now(),
		version:   version,
	}
}

func (h *Handler) GetTime(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()
	resp := map[string]interface{}{
		"utc":         now.Format(time.RFC3339),
		"iso8601":     now.Format("2006-01-02T15:04:05Z07:00"),
		"epoch":       now.Unix(),
		"nanoseconds": now.UnixNano(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) GetStatus(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"status":    "ok",
		"version":   h.version,
		"uptime":    time.Since(h.startTime).String(),
		"servers":   0,
		"reachable": 0,
		"timestamp": time.Now().UTC(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) GetServers(w http.ResponseWriter, r *http.Request) {
	servers := make([]interface{}, 0)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(servers)
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func (h *Handler) ReadinessCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"ready": "true"})
}

// Collector collects and exposes Prometheus metrics.
type Collector struct{}

func (c *Collector) WriteMetrics(w io.Writer) error {
	fmt.Fprintf(w, "# HELP timesphere_uptime_seconds Application uptime in seconds\n")
	fmt.Fprintf(w, "# TYPE timesphere_uptime_seconds gauge\n")
	fmt.Fprintf(w, "timesphere_uptime_seconds 0\n")
	return nil
}

// New creates a new server instance.
func New(cfg *Config, logger Logger, version string) (*Server, error) {
	monitor := &Monitor{}
	apiHandler := NewHandler(monitor, version)
	metricsCollector := &Collector{}

	return &Server{
		cfg:       cfg,
		monitor:   monitor,
		api:       apiHandler,
		metrics:   metricsCollector,
		logger:    logger,
		version:   version,
		startTime: time.Now(),
	}, nil
}

// Start begins the server.
func (s *Server) Start() error {
	// Start NTP monitoring
	s.monitor.Start()

	// Setup routes
	mux := http.NewServeMux()
	s.setupRoutes(mux)

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", s.cfg.Server.Host, s.cfg.Server.Port)
	s.httpSrv = &http.Server{
		Addr:         addr,
		Handler:      s.withLogging(s.withAuth(mux)),
		ReadTimeout:  time.Duration(s.cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(s.cfg.Server.WriteTimeout) * time.Second,
	}

	// Handle graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		s.logger.Info("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.cfg.Server.ShutdownTimeout)*time.Second)
		defer cancel()

		s.monitor.Stop()
		if err := s.httpSrv.Shutdown(ctx); err != nil {
			s.logger.Error("Server shutdown error", "error", err)
		}
	}()

	s.logger.Info("Starting TimeSphere", "version", s.version, "address", addr)
	return s.httpSrv.ListenAndServe()
}

// setupRoutes configures HTTP routes.
func (s *Server) setupRoutes(mux *http.ServeMux) {
	// API routes
	mux.HandleFunc("/api/time", s.api.GetTime)
	mux.HandleFunc("/api/status", s.api.GetStatus)
	mux.HandleFunc("/api/servers", s.api.GetServers)
	mux.HandleFunc("/health", s.api.HealthCheck)
	mux.HandleFunc("/ready", s.api.ReadinessCheck)
	mux.HandleFunc("/metrics", s.handleMetrics)

	// Curl-friendly endpoints
	mux.HandleFunc("/", s.handleRoot)
	mux.HandleFunc("/epoch", s.handleEpoch)
	mux.HandleFunc("/unix", s.handleUnix)
	mux.HandleFunc("/plain", s.handlePlain)

	// Static files
	staticContent, _ := fs.Sub(staticFS, "static")
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticContent))))

	// Dashboard
	mux.HandleFunc("/dashboard", s.handleDashboard)
}

// handleRoot handles the root endpoint with curl-friendly output.
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// Check if it's a browser request
	acceptHeader := r.Header.Get("Accept")
	userAgent := r.Header.Get("User-Agent")

	isBrowser := strings.Contains(acceptHeader, "text/html") ||
		strings.Contains(userAgent, "Mozilla") ||
		r.Method == "GET" && !strings.Contains(acceptHeader, "application/json")

	if isBrowser {
		s.handleDashboard(w, r)
		return
	}

	// For curl and other CLI tools, return plain ISO8601 time
	now := time.Now().UTC()
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprint(w, now.Format(time.RFC3339))
}

// handleEpoch returns Unix epoch timestamp.
func (s *Server) handleEpoch(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprint(w, now.Unix())
}

// handleUnix returns Unix time as JSON.
func (s *Server) handleUnix(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "{\"epoch\":%d,\"utc\":\"%s\"}\n", now.Unix(), now.Format(time.RFC3339))
}

// handlePlain returns plain text time.
func (s *Server) handlePlain(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprint(w, now.Format("2006-01-02T15:04:05Z"))
}

// handleMetrics serves Prometheus metrics.
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	if err := s.metrics.WriteMetrics(w); err != nil {
		s.logger.Error("Error writing metrics", "error", err)
	}
}

// handleDashboard serves the main dashboard.
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFS(templateFS, "templates/index.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		s.logger.Error("Template parse error", "error", err)
		return
	}

	data := map[string]interface{}{
		"Title":     s.cfg.UI.Title,
		"DarkMode":  s.cfg.UI.DarkMode,
		"Servers":   len(s.cfg.Servers),
		"Version":   s.version,
	}

	w.Header().Set("Content-Type", "text/html")
	if err := tmpl.Execute(w, data); err != nil {
		s.logger.Error("Template execution error", "error", err)
	}
}

// withAuth adds authentication middleware if configured.
func (s *Server) withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.cfg.Auth.Enabled {
			username, password, ok := r.BasicAuth()
			if !ok || username != s.cfg.Auth.Username || password != s.cfg.Auth.Password {
				w.Header().Set("WWW-Authenticate", `Basic realm="TimeSphere"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// withLogging adds request logging.
func (s *Server) withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		duration := time.Since(start)
		s.logger.Debug("Request", "method", r.Method, "path", r.URL.Path, "duration", duration.String())
	})
}
