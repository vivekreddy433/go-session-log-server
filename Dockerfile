# Use the latest Go 1.23 version
FROM golang:1.23

# Set working directory inside the container
WORKDIR /app

# Copy Go modules and install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire project into the container
COPY . .

# Build the Go application
RUN go build -o webhook-receiver ./cmd/main.go

# Expose the application port
EXPOSE 8080

# Run the application
CMD ["./webhook-receiver"]
