// Package metrics provides Prometheus-compatible metrics.
package metrics

import (
	"fmt"
	"io"
	"time"

	"timesphere/internal/ntp"
)

// Collector collects and exposes Prometheus metrics.
type Collector struct {
	monitor *ntp.Monitor
	startTime time.Time
}

// NewCollector creates a new metrics collector.
func NewCollector(monitor *ntp.Monitor) *Collector {
	return &Collector{
		monitor:   monitor,
		startTime: time.Now(),
	}
}

// WriteMetrics writes Prometheus-formatted metrics to the writer.
func (c *Collector) WriteMetrics(w io.Writer) error {
	statuses := c.monitor.GetStatuses()

	// Uptime metric
	uptime := time.Since(c.startTime).Seconds()
	fmt.Fprintf(w, "# HELP timesphere_uptime_seconds Application uptime in seconds\n")
	fmt.Fprintf(w, "# TYPE timesphere_uptime_seconds gauge\n")
	fmt.Fprintf(w, "timesphere_uptime_seconds %.2f\n", uptime)

	// Server count metrics
	totalServers := len(statuses)
	reachableServers := 0
	for _, s := range statuses {
		if s.Reachable {
			reachableServers++
		}
	}

	fmt.Fprintf(w, "# HELP timesphere_servers_total Total number of configured servers\n")
	fmt.Fprintf(w, "# TYPE timesphere_servers_total gauge\n")
	fmt.Fprintf(w, "timesphere_servers_total %d\n", totalServers)

	fmt.Fprintf(w, "# HELP timesphere_servers_reachable Number of reachable servers\n")
	fmt.Fprintf(w, "# TYPE timesphere_servers_reachable gauge\n")
	fmt.Fprintf(w, "timesphere_servers_reachable %d\n", reachableServers)

	// Per-server metrics
	fmt.Fprintf(w, "# HELP timesphere_server_offset_nanoseconds Time offset from server in nanoseconds\n")
	fmt.Fprintf(w, "# TYPE timesphere_server_offset_nanoseconds gauge\n")

	fmt.Fprintf(w, "# HELP timesphere_server_delay_nanoseconds Round trip delay in nanoseconds\n")
	fmt.Fprintf(w, "# TYPE timesphere_server_delay_nanoseconds gauge\n")

	fmt.Fprintf(w, "# HELP timesphere_server_jitter_nanoseconds Estimated jitter in nanoseconds\n")
	fmt.Fprintf(w, "# TYPE timesphere_server_jitter_nanoseconds gauge\n")

	fmt.Fprintf(w, "# HELP timesphere_server_stratum Server stratum level\n")
	fmt.Fprintf(w, "# TYPE timesphere_server_stratum gauge\n")

	fmt.Fprintf(w, "# HELP timesphere_server_reachable Server reachability status\n")
	fmt.Fprintf(w, "# TYPE timesphere_server_reachable gauge\n")

	for _, s := range statuses {
		labels := fmt.Sprintf("{name=\"%s\",address=\"%s\"}", s.Name, s.Address)
		
		offset := int64(0)
		delay := int64(0)
		jitter := int64(0)
		stratum := int(0)
		reachable := 0

		if s.Reachable {
			offset = s.Offset.Nanoseconds()
			delay = s.Delay.Nanoseconds()
			jitter = s.Jitter.Nanoseconds()
			stratum = int(s.Stratum)
			reachable = 1
		}

		fmt.Fprintf(w, "timesphere_server_offset_nanoseconds%s %d\n", labels, offset)
		fmt.Fprintf(w, "timesphere_server_delay_nanoseconds%s %d\n", labels, delay)
		fmt.Fprintf(w, "timesphere_server_jitter_nanoseconds%s %d\n", labels, jitter)
		fmt.Fprintf(w, "timesphere_server_stratum%s %d\n", labels, stratum)
		fmt.Fprintf(w, "timesphere_server_reachable%s %d\n", labels, reachable)
	}

	return nil
}
