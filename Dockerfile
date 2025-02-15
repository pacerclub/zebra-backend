FROM golang:1.21-alpine

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Create migrations directory
RUN mkdir -p /app/internal/db/migrations

# Copy source code and migrations
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -o main ./cmd/api

# Use a smaller image for the final container
FROM alpine:latest

WORKDIR /app

# Create migrations directory in final image
RUN mkdir -p /app/internal/db/migrations

# Copy the binary and required files from builder
COPY --from=0 /app/main .
COPY --from=0 /app/.env .
COPY --from=0 /app/internal/db/migrations/* ./internal/db/migrations/

# Expose the application port
EXPOSE 8080

# Run the application
CMD ["./main"]
