# TimeSphere - Enterprise Time Monitoring
# Single container, minimal footprint

FROM golang:1.19-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o timesphere ./cmd/timesphere

# Runtime stage
FROM alpine:3.18

# Install CA certificates for HTTPS
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 timesphere && \
    adduser -D -u 1000 -G timesphere timesphere

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/timesphere .

# Create directories
RUN mkdir -p /storage /etc/timesphere && \
    chown -R timesphere:timesphere /app /storage /etc/timesphere

# Copy default config if exists
COPY --chown=timesphere:timesphere config.yaml.example /etc/timesphere/config.yaml

USER timesphere

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget -q --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/timesphere"]
CMD ["-config", "/etc/timesphere/config.yaml"]
