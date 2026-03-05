FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /tenant-gateway ./cmd/gateway

# Final stage
FROM alpine:3.21

RUN apk add --no-cache ca-certificates

COPY --from=builder /tenant-gateway /usr/local/bin/tenant-gateway

EXPOSE 8080

ENTRYPOINT ["tenant-gateway"]
