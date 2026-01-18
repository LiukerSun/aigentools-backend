# Build stage
FROM golang:1.25-alpine AS builder

# Set environment variables
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    GOPROXY=https://goproxy.cn,direct

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN go build -o main .

# Run stage
FROM alpine:latest

# Install certificates for HTTPS and timezone data
RUN apk --no-cache add ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/main .

# Copy .env.example as .env if .env doesn't exist (optional, usually handled by volume or env_file in compose)
# COPY .env.example .env

# Create logs directory and set permissions
RUN mkdir -p logs && chmod 777 logs

# Create a non-root user
RUN adduser -D -g '' appuser
USER appuser

# Expose port
EXPOSE 8080

# Command to run the executable
CMD ["./main"]