# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod ./

# Copy all source code (needed for go mod tidy to work)
COPY *.go ./
COPY static/ ./static/
COPY locales/ ./locales/

# Download dependencies and generate go.sum
RUN go mod tidy && go mod download

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o sleep-proxy .

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/sleep-proxy .

# Expose the default proxy port
EXPOSE 8000

CMD ["./sleep-proxy"]
