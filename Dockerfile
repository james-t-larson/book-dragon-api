# ==========================================
# Step 1: Build the Application
# ==========================================
FROM golang:1.25-bookworm AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy the go.mod and go.sum files first.
# Doing this before copying the rest of the code caches your dependencies 
# so they don't redownload every time you change a line of Go code.
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the Go application.
# CGO_ENABLED=1 is absolutely required here for the SQLite driver to compile.
RUN CGO_ENABLED=1 GOOS=linux go build -a -o book-dragon-api ./cmd/api/main.go

# ==========================================
# Step 2: Create the Production Image
# ==========================================
FROM debian:bookworm-slim

# Install CA certificates (needed if your API makes HTTPS requests) 
# and clean up the apt cache to keep the image size down
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy the finished binary from the builder stage above
COPY --from=builder /app/book-dragon-api /usr/local/bin/book-dragon-api

# Create the data directory where your SQLite database will live.
# This matches the volume we set up in the docker-compose.yml file.
RUN mkdir -p /app/data

# Expose port 8080 to the Docker network
EXPOSE 8080

# Run the API
CMD ["book-dragon-api"]
