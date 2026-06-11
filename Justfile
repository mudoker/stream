# Justfile for stream

# Build the application
build:
    go build -o stream main.go

# Run all tests with coverage
test:
    go test -v -coverpkg=./... ./...

# Run the application
run:
    go run main.go

# Make the local binary executable
apply: build
    chmod +x stream
