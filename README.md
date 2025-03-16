# Webhook Receiver

The Webhook Receiver is a robust Golang-based API designed to receive JSON payloads, temporarily store them in memory, and forward them in batches based on configurable size and time constraints. It ensures reliable delivery through intelligent retry mechanisms and provides comprehensive logging for operational visibility.


### Prerequisites

Before running the project, make sure you have:

- Go 1.22+ installed 
- Docker (optional, for containerized execution)
- Postman (optional, for API testing)

### Running the Application Locally

#### 1. Clone the Repository

git clone https://github.com/vivekreddy433/go-session-log-server.git

#### 2. Set Up Environment Variables

Create a ".env" file in the project root and add:


BATCH_SIZE=5
BATCH_INTERVAL=10
POST_ENDPOINT=https://webhook.site/335d0d0d-3883-4248-9926-820357e468a2
LOG_LEVEL=INFO
LOG_FORMAT=JSON

#### 3. Install Dependencies and Run


go mod tidy
go run cmd/main.go

## Testing the API

### Health Check


curl -X GET http://localhost:8080/healthz

Expected Response:

OK

### Send a Webhook Log

curl -X POST http://localhost:8080/log \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 1,
    "total": 2.50,
    "title": "Test Log",
    "meta": {
      "logins": [{"time": "2022-01-01T00:00:00Z", "ip": "192.168.1.1"}],
      "phone_numbers": {"home": "555-1212", "mobile": "123-5555"}
    },
    "completed": false
  }'

## Running Unit Tests

Run tests with coverage:

go test -coverprofile=coverage.out ./...

Generate an HTML coverage report:

go tool cover -html=coverage.out -o coverage.html


Open `coverage.html` in a browser to check test coverage per file.

## Running with Docker

### 1. Build the Docker Image


docker build -t webhook-receiver .

### 2. Run the Container


docker run --env-file=.env -p 8080:8080 webhook-receiver

##  Configuration Options

BATCH_SIZE=5
BATCH_INTERVAL=10
POST_ENDPOINT=https://webhook.site/335d0d0d-3883-4248-9926-820357e468a2
LOG_LEVEL=INFO
LOG_FORMAT=JSON


### Linting

The project uses [golangci-lint](https://github.com/golangci/golangci-lint) for linting:

golangci-lint run