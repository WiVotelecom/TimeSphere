# TimeSphere

**Enterprise Time Monitoring Dashboard** - A lightweight, self-hosted NTP monitoring solution with a modern web interface.

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go Version](https://img.shields.io/badge/go-1.19+-blue.svg)
![Docker](https://img.shields.io/badge/docker-ready-green.svg)

## Overview

TimeSphere is a comprehensive NTP (Network Time Protocol) monitoring tool that helps you track time synchronization across multiple servers. It provides real-time metrics, alerting capabilities, and an intuitive web dashboard for monitoring your infrastructure's time accuracy.

## Features

- 🕐 **Real-time NTP Monitoring** - Query multiple NTP servers simultaneously
- 📊 **Interactive Dashboard** - Modern web UI with analog/digital clocks and metrics visualization
- 🌙 **Dark/Light Mode** - Toggle between themes based on preference
- ⚠️ **Alerting System** - Configurable thresholds for offset, jitter, and reachability
- 📈 **Historical Data** - SQLite-based storage for trend analysis
- 🔒 **Authentication** - Optional user authentication for secure deployments
- 🔐 **TLS Support** - HTTPS encryption for production environments
- 🐳 **Docker Ready** - Single-container deployment with minimal footprint
- 🔍 **Auto-discovery** - Optional network scanning for local NTP servers
- 📡 **Webhook Integration** - Send alerts to external systems

## Quick Start

### Docker Compose (Recommended)

```bash
# Clone the repository
git clone https://github.com/your-org/timesphere.git
cd timesphere

# Copy and customize configuration
cp config.yaml.example config.yaml

# Start with Docker Compose
docker-compose up -d

# Access the dashboard
open http://localhost:8080
```

### Docker Run

```bash
docker run -d \
  --name timesphere \
  -p 8080:8080 \
  -v $(pwd)/config.yaml:/etc/timesphere/config.yaml:ro \
  -v timesphere-data:/storage \
  timesphere:latest
```

### Build from Source

```bash
# Prerequisites: Go 1.19+, GCC, musl-dev

# Clone and build
git clone https://github.com/your-org/timesphere.git
cd timesphere

# Download dependencies
go mod download

# Build binary
CGO_ENABLED=0 go build -ldflags="-s -w" -o timesphere ./cmd/timesphere

# Run
./timesphere -config /path/to/config.yaml
```

## Configuration

Configuration is managed via YAML file (default: `/etc/timesphere/config.yaml`).

### Example Configuration

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: 15
  write_timeout: 15
  shutdown_timeout: 30

# NTP Servers to monitor
servers:
  - name: "pool.ntp.org"
    address: "pool.ntp.org"
    port: 123
    enabled: true
  - name: "time.google.com"
    address: "time.google.com"
    port: 123
    enabled: true

# Alerting thresholds
thresholds:
  offset_ms: 20      # Alert if offset exceeds 20ms
  jitter_ms: 10      # Alert if jitter exceeds 10ms
  poll_interval: 64  # Poll interval in seconds
  reach_timeout: 3   # Consecutive failures before alert

# UI settings
ui:
  title: "TimeSphere"
  dark_mode: true
  refresh_rate: 1    # Refresh rate in seconds

# Authentication (optional)
auth:
  enabled: false
  username: "admin"
  password: "changeme"

# TLS (optional)
tls:
  enabled: false
  cert_file: "/path/to/cert.pem"
  key_file: "/path/to/key.pem"

# History storage
history:
  enabled: true
  sample_rate_1h: 10  # Samples per hour for historical data

# Alerting webhook (optional)
alerting:
  enabled: false
  webhook: "https://hooks.slack.com/..."

# Auto-discovery (optional)
discovery:
  enabled: false
  cidrs: ["192.168.1.0/24"]

# Logging
logging:
  level: "info"       # debug, info, warn, error
  format: "json"      # json or text
```

## API Endpoints

TimeSphere exposes a RESTful API for integration:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check endpoint |
| `/api/status` | GET | Current NTP server status |
| `/api/metrics` | GET | Detailed metrics data |
| `/api/history` | GET | Historical data |
| `/api/servers` | GET | List configured servers |

### Example API Response

```json
{
  "servers": [
    {
      "name": "pool.ntp.org",
      "address": "pool.ntp.org",
      "offset": "2.5ms",
      "delay": "15.3ms",
      "jitter": "1.2ms",
      "reachable": true,
      "stratum": 2,
      "last_query": "2024-01-15T10:30:00Z"
    }
  ],
  "system_time": "2024-01-15T10:30:00Z",
  "status": "healthy"
}
```

## Project Structure

```
timesphere/
├── cmd/                    # Application entry point
│   └── timesphere/
│       └── main.go
├── internal/
│   ├── api/                # HTTP API handlers
│   ├── config/             # Configuration parsing
│   ├── metrics/            # Metrics collection
│   ├── ntp/                # NTP client implementation
│   ├── server/             # HTTP server setup
│   └── storage/            # SQLite storage layer
├── web/
│   └── templates/          # HTML templates
├── config.yaml.example     # Example configuration
├── docker-compose.yml      # Docker Compose setup
├── Dockerfile              # Container build instructions
└── go.mod                  # Go module definition
```

## Security Considerations

- **Non-root container**: The Docker image runs as a non-root user (UID 1000)
- **Read-only filesystem**: Container filesystem is read-only except for data volumes
- **Security options**: `no-new-privileges` is enabled by default
- **Health checks**: Built-in health monitoring for container orchestration
- **Optional authentication**: Enable auth for production deployments
- **TLS support**: Encrypt traffic with HTTPS

## Monitoring & Observability

- **Health endpoint**: `/health` for load balancer and orchestrator checks
- **Structured logging**: JSON-formatted logs for log aggregation systems
- **Metrics export**: API endpoints for integration with monitoring tools
- **Alert webhooks**: Push notifications to Slack, PagerDuty, or custom endpoints

## Development

### Running Tests

```bash
go test ./...
```

### Building Docker Image

```bash
docker build -t timesphere:latest .
```

### Local Development

```bash
# Run with hot reload (requires air or similar)
go run ./cmd/timesphere -config config.yaml

# Or build and run
go build -o timesphere ./cmd/timesphere
./timesphere -config config.yaml
```

## Troubleshooting

### Common Issues

**Container won't start:**
- Check config.yaml syntax
- Ensure port 8080 is not in use
- Verify volume permissions

**NTP servers unreachable:**
- Check firewall rules (UDP port 123)
- Verify DNS resolution
- Test connectivity: `ntpdate -q pool.ntp.org`

**High offset values:**
- Check system time configuration
- Verify hardware clock
- Consider using stratum 1 servers

### Logs

View container logs:
```bash
docker logs timesphere
```

Enable debug logging in config:
```yaml
logging:
  level: "debug"
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- NTP protocol implementation based on RFC 5905
- Inspired by enterprise monitoring solutions
- Built with Go and modern web technologies

## Support

For issues, questions, or contributions:
- Open an issue on GitHub
- Check existing documentation
- Review configuration examples

---

**TimeSphere** - Keeping your infrastructure in sync, one packet at a time. 🕐
