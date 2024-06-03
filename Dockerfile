# Stage 1: Build the Go app
FROM golang:latest AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy the current directory contents into the container at /app
COPY . .

# Build the Go app
RUN go build -o main .

# Stage 2: Create a lightweight runtime image
FROM alpine:latest

# Set the working directory inside the container
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/main .

# Install necessary dependencies
RUN apk --no-cache add ca-certificates

# Command to run the executable
CMD ["./main"]
