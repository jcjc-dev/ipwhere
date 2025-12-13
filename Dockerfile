# ============================================
# Stage 1: Build Frontend (Node.js + Tailwind)
# ============================================
FROM node:20-alpine AS frontend-builder

WORKDIR /app/web

# Install dependencies first (better caching)
COPY web/package.json web/package-lock.json* ./
RUN npm ci || npm install

# Copy source and build
COPY web/ ./
RUN npm run build

# ============================================
# Stage 2: Build Go Binary
# ============================================
FROM golang:1.24-alpine AS go-builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Download Go dependencies first (better caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Copy built frontend assets from Stage 1
COPY --from=frontend-builder /app/web/dist ./cmd/ipwhere/static/

# Ensure data directory exists (databases will be copied at runtime stage)
RUN mkdir -p /app/data

# Build with optimizations for multiple architectures
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ENV CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH}
RUN go build -ldflags="-w -s" -o /app/ipwhere ./cmd/ipwhere

# ============================================
# Stage 3: Runtime (Minimal Image)
# ============================================
FROM alpine:3.19 AS runtime

# Add ca-certificates for HTTPS and timezone data
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy binary from build stage
COPY --from=go-builder /app/ipwhere .

# Copy MMDB databases from build context (downloaded by Makefile)
# This avoids downloading per-architecture in multi-arch builds
COPY data/*.mmdb ./data/

# Set ownership
RUN chown -R appuser:appgroup /app

# Use non-root user
USER appuser

# Expose port
EXPOSE 8080

# Environment variables
ENV LISTEN_ADDR=:8080 \
    CITY_DB_PATH=/app/data/dbip-city-lite.mmdb \
    ASN_DB_PATH=/app/data/dbip-asn-lite.mmdb

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s \
    CMD wget --quiet --tries=1 --spider http://localhost:8080/health || exit 1

# Run
ENTRYPOINT ["./ipwhere"]
CMD ["-l", ":8080"]
