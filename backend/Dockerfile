# Production Dockerfile for Bible App Backend
# Build stage
FROM golang:1.21-alpine AS builder

# Install necessary build tools
RUN apk add --no-cache git ca-certificates tzdata && \
    update-ca-certificates

WORKDIR /app

# Copy go.mod and go.sum files first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build a static binary (no C dependencies)
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static" -s -w' -o bibleapp cmd/server/main.go

# Final stage with minimal image
FROM alpine:3.19

# Add required runtime dependencies
RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -H -h /app appuser

WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/bibleapp .

# Use non-root user for better security
USER appuser

# Expose port - default to 10000 for Render
EXPOSE 10000

# Run the compiled binary (not 'go run')
CMD ["./bibleapp"]
