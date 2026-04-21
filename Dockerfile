# syntax=docker/dockerfile:1

# Builder stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go.mod first for better caching
COPY go.mod ./

# Download dependencies
RUN go mod download || true

# Copy source code
COPY . .

# Ensure dependencies are tidy (generates go.sum if missing)
RUN go mod tidy

# Build the application
RUN go build -o main .

# Final stage
FROM alpine:latest

# 设置 UTF-8 环境，避免中文乱码
ENV LANG=C.UTF-8
ENV LC_ALL=C.UTF-8

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/main .

# Copy static files (management panel)
COPY --from=builder /app/public ./public

# Expose port
EXPOSE 3000

# Command to run the application
CMD ["./main"]
