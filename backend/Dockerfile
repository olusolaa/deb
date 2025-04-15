# Simple development Dockerfile for Bible App Backend
FROM golang:1.21

WORKDIR /app

# Copy go.mod and go.sum files first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Expose the port the app listens on
EXPOSE 8084

# Run the application
CMD ["go", "run", "cmd/server/main.go"]
