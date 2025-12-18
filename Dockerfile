# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o apiserver ./cmd/apiserver

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/apiserver .

# Expose port
EXPOSE 8080

# Set environment variables with defaults
ENV MONGO_URI=mongodb://localhost:27017
ENV SERVER_ADDRESS=:8080

# Run the binary
CMD ["./apiserver"]