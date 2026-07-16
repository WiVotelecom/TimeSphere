// Package config handles application configuration loading and validation.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Server represents an NTP server to monitor.
type Server struct {
	Name    string `yaml:"name" json:"name"`
	Address string `yaml:"address" json:"address"`
	Port    int    `yaml:"port,omitempty" json:"port,omitempty"`
	Enabled bool   `yaml:"enabled" json:"enabled"`
}

// Thresholds define alerting thresholds.
type Thresholds struct {
	OffsetMS      int64 `yaml:"offset_ms" json:"offset_ms"`           // Milliseconds
	JitterMS      int64 `yaml:"jitter_ms" json:"jitter_ms"`           // Milliseconds
	PollInterval  int   `yaml:"poll_interval" json:"poll_interval"`   // Seconds
	ReachTimeout  int   `yaml:"reach_timeout" json:"reach_timeout"`   // Number of failed polls
}

// UIConfig contains UI-related settings.
type UIConfig struct {
	Title       string `yaml:"title" json:"title"`
	DarkMode    bool   `yaml:"dark_mode" json:"dark_mode"`
	RefreshRate int    `yaml:"refresh_rate" json:"refresh_rate"` // Seconds
}

// AuthConfig contains authentication settings.
type AuthConfig struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
}

// TLSConfig contains TLS settings.
type TLSConfig struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`
	CertFile string `yaml:"cert_file" json:"cert_file"`
	KeyFile  string `yaml:"key_file" json:"key_file"`
}

// HistoryConfig contains history retention settings.
type HistoryConfig struct {
	Enabled       bool          `yaml:"enabled" json:"enabled"`
	Retention1h   time.Duration `yaml:"retention_1h" json:"retention_1h"`
	Retention24h  time.Duration `yaml:"retention_24h" json:"retention_24h"`
	Retention7d   time.Duration `yaml:"retention_7d" json:"retention_7d"`
	Retention30d  time.Duration `yaml:"retention_30d" json:"retention_30d"`
	SampleRate1h  int           `yaml:"sample_rate_1h" json:"sample_rate_1h"`   // Seconds between samples
	SampleRate24h int           `yaml:"sample_rate_24h" json:"sample_rate_24h"` // Seconds between samples
	SampleRate7d  int           `yaml:"sample_rate_7d" json:"sample_rate_7d"`   // Seconds between samples
}

// AlertingConfig contains alerting settings.
type AlertingConfig struct {
	Enabled bool     `yaml:"enabled" json:"enabled"`
	Webhook string   `yaml:"webhook,omitempty" json:"webhook,omitempty"`
	Email   string   `yaml:"email,omitempty" json:"email,omitempty"`
	Syslog  string   `yaml:"syslog,omitempty" json:"syslog,omitempty"`
}

// DiscoveryConfig contains auto-discovery settings.
type DiscoveryConfig struct {
	Enabled bool     `yaml:"enabled" json:"enabled"`
	CIDRs   []string `yaml:"cidrs" json:"cidrs"`
}

// Config is the main application configuration.
type Config struct {
	Server     ServerConfig     `yaml:"server" json:"server"`
	Servers    []Server         `yaml:"servers" json:"servers"`
	Thresholds Thresholds       `yaml:"thresholds" json:"thresholds"`
	UI         UIConfig         `yaml:"ui" json:"ui"`
	Auth       AuthConfig       `yaml:"auth" json:"auth"`
	TLS        TLSConfig        `yaml:"tls" json:"tls"`
	History    HistoryConfig    `yaml:"history" json:"history"`
	Alerting   AlertingConfig   `yaml:"alerting" json:"alerting"`
	Discovery  DiscoveryConfig  `yaml:"discovery" json:"discovery"`
	Logging    LoggingConfig    `yaml:"logging" json:"logging"`
}

// ServerConfig contains server settings.
type ServerConfig struct {
	Port            int    `yaml:"port" json:"port"`
	Host            string `yaml:"host" json:"host"`
	ReadTimeout     int    `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout    int    `yaml:"write_timeout" json:"write_timeout"`
	ShutdownTimeout int    `yaml:"shutdown_timeout" json:"shutdown_timeout"`
}

// LoggingConfig contains logging settings.
type LoggingConfig struct {
	Level  string `yaml:"level" json:"level"`
	Format string `yaml:"format" json:"format"` // json or text
}

// DefaultConfig returns a configuration with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:            8080,
			Host:            "0.0.0.0",
			ReadTimeout:     15,
			WriteTimeout:    15,
			ShutdownTimeout: 30,
		},
		Servers: []Server{},
		Thresholds: Thresholds{
			OffsetMS:      20,
			JitterMS:      10,
			PollInterval:  64,
			ReachTimeout:  3,
		},
		UI: UIConfig{
			Title:       "TimeSphere",
			DarkMode:    true,
			RefreshRate: 1,
		},
		Auth: AuthConfig{
			Enabled: false,
		},
		TLS: TLSConfig{
			Enabled: false,
		},
		History: HistoryConfig{
			Enabled:       true,
			Retention1h:   time.Hour,
			Retention24h:  24 * time.Hour,
			Retention7d:   7 * 24 * time.Hour,
			Retention30d:  30 * 24 * time.Hour,
			SampleRate1h:  10,
			SampleRate24h: 60,
			SampleRate7d:  300,
		},
		Alerting: AlertingConfig{
			Enabled: false,
		},
		Discovery: DiscoveryConfig{
			Enabled: false,
			CIDRs:   []string{},
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
	}
}

// Load reads configuration from a YAML file.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	return cfg, nil
}

// Save writes configuration to a YAML file.
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}

// ToJSON converts config to JSON string.
func (c *Config) ToJSON() ([]byte, error) {
	return json.MarshalIndent(c, "", "  ")
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Server.Port)
	}

	if c.Logging.Level != "" {
		validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
		if !validLevels[c.Logging.Level] {
			return fmt.Errorf("invalid log level: %s", c.Logging.Level)
		}
	}

	return nil
}
