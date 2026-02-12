# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o operator ./cmd/operator

# Final stage
FROM alpine:3.19

# Add non-root user for security
RUN adduser -D -g '' appuser

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/operator .

# Use non-root user
USER appuser

# Expose ports
EXPOSE 8080 8081

# Run the operator
ENTRYPOINT ["./operator"]
