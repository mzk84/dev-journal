# --- Stage 1: Build ---
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install git which is needed for go modules and for the app to run
RUN apk add --no-cache git gcc libc-dev

# Copy go mod and sum files
COPY go.mod go.sum ./
# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code
COPY . .

# Build the application
# CGO_ENABLED is needed for go-sqlite3
# -ldflags="-w -s" strips debug information, reducing binary size
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o dev-journal ./cmd/app

# --- Stage 2: Final Image ---
FROM alpine:latest

WORKDIR /app

# We need git for cloning/pulling the repo and sqlite for the db driver
RUN apk add --no-cache git sqlite

# Copy built binary from the builder stage
COPY --from=builder /app/gmd-site .

# Copy web assets
COPY web ./web

# Create a non-root user for security
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
RUN chown -R appuser:appgroup /app
USER appuser

# Expose port 8080
EXPOSE 8080

# The command to run the application
CMD ["./dev-journal"]
