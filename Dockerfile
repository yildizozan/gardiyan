FROM golang:1.23.12-alpine AS builder

WORKDIR /app

# Copy Go module files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o gardiyan main.go

# Final stage - wolfi image
FROM cgr.dev/chainguard/wolfi-base:latest

# Create non-root user in wolfi
RUN adduser -D -g '' gardiyan

# Copy binary
COPY --from=builder /app/gardiyan /usr/local/bin/gardiyan

# Use non-root user
USER gardiyan

# Expose port
EXPOSE 8080

# Run when container starts
CMD ["gardiyan"]
