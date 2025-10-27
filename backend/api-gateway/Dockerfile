# =============================================================================
# NUCLEAR AO3 API GATEWAY DOCKERFILE
# High-performance unified GraphQL + REST API for all microservices
# =============================================================================

FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy shared libraries first (for better caching)
COPY shared/ ./shared/

# Copy API Gateway source code
COPY api-gateway/ ./api-gateway/

# Set working directory to API Gateway
WORKDIR /app/api-gateway

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o api-gateway .

# =============================================================================
# PRODUCTION IMAGE
# =============================================================================

FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata curl

# Create non-root user
RUN addgroup -g 1000 appgroup && \
    adduser -D -s /bin/sh -u 1000 -G appgroup appuser

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/api-gateway/api-gateway .

# Copy shared assets if needed
COPY --from=builder /app/shared ./shared

# Change ownership to non-root user
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Run the API Gateway
CMD ["./api-gateway"]