// Package ntp provides NTP monitoring engine.
package ntp

import (
	"sync"
	"time"

	"timesphere/internal/config"
)

// Monitor manages NTP server monitoring.
type Monitor struct {
	mu       sync.RWMutex
	servers  []config.Server
	statuses map[string]*ServerStatus
	interval time.Duration
	timeout  time.Duration
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

// NewMonitor creates a new NTP monitor.
func NewMonitor(servers []config.Server, interval, timeout time.Duration) *Monitor {
	return &Monitor{
		servers:  servers,
		statuses: make(map[string]*ServerStatus),
		interval: interval,
		timeout:  timeout,
		stopCh:   make(chan struct{}),
	}
}

// Start begins the monitoring loop.
func (m *Monitor) Start() {
	m.wg.Add(1)
	go m.run()
}

// Stop halts the monitoring loop.
func (m *Monitor) Stop() {
	close(m.stopCh)
	m.wg.Wait()
}

// run is the main monitoring loop.
func (m *Monitor) run() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	// Initial poll
	m.pollAll()

	for {
		select {
		case <-ticker.C:
			m.pollAll()
		case <-m.stopCh:
			return
		}
	}
}

// pollAll queries all configured servers.
func (m *Monitor) pollAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, srv := range m.servers {
		if !srv.Enabled {
			continue
		}

		status, _ := QueryNTP(srv.Address, srv.Port, m.timeout)
		status.Name = srv.Name
		m.statuses[srv.Address] = status
	}
}

// GetStatuses returns all server statuses.
func (m *Monitor) GetStatuses() map[string]*ServerStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*ServerStatus)
	for k, v := range m.statuses {
		result[k] = v
	}
	return result
}

// GetStatus returns status for a specific server.
func (m *Monitor) GetStatus(address string) (*ServerStatus, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status, ok := m.statuses[address]
	return status, ok
}

// UpdateServers updates the list of monitored servers.
func (m *Monitor) UpdateServers(servers []config.Server) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.servers = servers
}
