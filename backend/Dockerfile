# Stage 1: Build the Go binary
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum files first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the Go app - Ensure the output path matches the main func location
# Statically link CGO_ENABLED=0 for smaller image, strip symbols with -w -s
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/server ./cmd/server/main.go

# Stage 2: Create the final minimal image
FROM gcr.io/distroless/static-debian11

WORKDIR /app

# Copy the built binary from the builder stage
COPY --from=builder /app/server /app/server

# Copy the .env file (Alternatively, rely solely on Cloud Run Env Vars/Secrets)
# COPY .env .env # Comment this out if using Cloud Run secrets exclusively

# Expose the port the app listens on (should match config/viper default/env var)
# This DOES NOT publish the port, Cloud Run handles that.
EXPOSE 8080

# Command to run the executable
# The binary needs to listen on the port specified by the PORT env var,
# which Cloud Run provides automatically (default 8080).
# Viper should pick up the PORT env var set by Cloud Run.
CMD ["/app/server"]
