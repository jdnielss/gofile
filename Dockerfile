# Stage 1: Build the Go application
FROM golang:latest AS builder

# Set the working directory inside the builder container
WORKDIR /app

# Copy go.mod and go.sum files
COPY . .

# Download dependencies
RUN go mod download

# Copy the entire source code
COPY . .

# Build the Go application
RUN go build -o myapp .

# Stage 2: Create a smaller runtime image
FROM alpine:latest

# Set the working directory inside the runtime container
WORKDIR /app

# Copy the binary from the builder container to the runtime container
COPY --from=builder /app/myapp .

# Command to run the executable
CMD ["./myapp"]
