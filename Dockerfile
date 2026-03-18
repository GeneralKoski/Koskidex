FROM golang:1.23-alpine AS builder

WORKDIR /app

# Download dependencies
COPY go.mod ./
# COPY go.sum ./ # If go.sum exists
RUN go mod download

# Copy source files
COPY . .

# Build binary
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -ldflags="-w -s -X main.version=${VERSION}" -o koskidex main.go

# Final slim image
FROM alpine:3.19

# Create necessary directories
RUN mkdir -p /data

# Copy binary from builder
COPY --from=builder /app/koskidex /usr/local/bin/

# Expose HTTP port
EXPOSE 7700

# Volume for persistence
VOLUME /data

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:7700/health || exit 1

# Default entrypoint
ENTRYPOINT ["koskidex", "--data-dir", "/data"]
