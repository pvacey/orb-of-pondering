# Stage 1: Build the application
FROM golang:1.25.5-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum files and download dependencies for efficient caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the static binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix netgo -o main .

# Stage 2: Create the final, minimal image
FROM alpine

# Copy the built binary from the 'builder' stage
COPY --from=builder /app/main /

# Set the entrypoint command to run the executable
CMD ["/main", "--ssh"]

