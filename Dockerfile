# Build stage
FROM golang:alpine AS builder

# Install build dependencies for CGO and Node.js for frontend build
RUN apk add --no-cache gcc musl-dev nodejs npm

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build frontend first
WORKDIR /app/webapp-react
RUN npm ci && npm run build

# Build the application
WORKDIR /app
RUN CGO_ENABLED=1 GOOS=linux go build -o server ./cmd/server

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/server .

# Copy static files
COPY --from=builder /app/webapp ./webapp
COPY --from=builder /app/webapp-react ./webapp-react
COPY --from=builder /app/res ./res

# Expose port
EXPOSE 10000

# Run the server
CMD ["./server"]
