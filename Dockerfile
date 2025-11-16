# --- Build Stage ---
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy module files and download dependencies
# This leverages Docker cache layers
COPY go.mod go.sum./
RUN go mod download

# Copy the rest of the source code
COPY..

# Build the application
# CGO_ENABLED=0 disables CGO for a static binary
# -ldflags="-w -s" strips debugging symbols to reduce binary size
RUN CGO_ENABLED=0 GOOS=linux go build \
    -a -installsuffix cgo \
    -ldflags="-w -s" \
    -o /heimdall./main.go

# --- Final Stage ---
FROM alpine:latest

# Add Certificate Authorities for making HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the built binary from the builder stage
COPY --from=builder /heimdall.

# Expose the port the app runs on
EXPOSE 8080

# Run the binary
ENTRYPOINT ["./heimdall"]
