# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY *.go ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o service-port-monitor .

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/service-port-monitor .

# Copy default configuration (can be overridden with volume mount)
COPY targets.txt .

# Create volume for logs
VOLUME ["/app/logs"]

# Set log file to logs directory
ENV LOG_FILE=/app/logs/checker.log
ENV CONFIG_FILE=/app/targets.txt

# Run the application
CMD ["./service-port-monitor"]
