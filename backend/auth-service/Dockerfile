# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy shared modules
COPY shared/ ./shared/

# Copy source code
COPY auth-service/ ./auth-service/

# Build the application
WORKDIR /app/auth-service
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/auth-service/main .

# Copy shared modules for runtime (if needed)
COPY --from=builder /app/shared ./shared

# Expose port
EXPOSE 8081

# Run the application
CMD ["./main"]