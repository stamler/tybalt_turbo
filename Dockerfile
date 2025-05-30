# Multi-stage Dockerfile for Tybalt PocketBase Application

# Stage 1: Build the Svelte UI
FROM node:20-alpine AS ui-builder

WORKDIR /app/ui

# Copy package files
COPY ui/package.json ui/package-lock.json ./

# Install dependencies
RUN npm ci

# Copy UI source code
COPY ui/ ./

# Build the UI
RUN npm run build

# Stage 2: Build the Go application
FROM golang:1.23-alpine AS go-builder

# Install git and other build dependencies
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy go mod files
COPY app/go.mod app/go.sum ./

# Download dependencies
RUN go mod download

# Copy Go source code
COPY app/ ./

# Copy built UI from previous stage
COPY --from=ui-builder /app/ui/build ./pb_public

# Build the Go application
RUN CGO_ENABLED=1 go build -o tybalt main.go

# Stage 3: Final runtime image
FROM alpine:latest

# Install ca-certificates for HTTPS support
RUN apk add --no-cache ca-certificates

# Create app directory
WORKDIR /pb

# Copy the built application and UI
COPY --from=go-builder /app/tybalt ./tybalt

# Copy the UI assets
COPY --from=go-builder /app/pb_public ./pb_public

# Create pb_data directory for database persistence
RUN mkdir -p /pb/pb_data

# Expose port 8080
EXPOSE 8080

# Start the application
CMD ["./tybalt", "serve", "--http=0.0.0.0:8080"] 