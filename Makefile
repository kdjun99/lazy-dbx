.PHONY: build run test lint lint-fix fmt vet clean

# Build
build:
	go build -o bin/lazy-dbx .

run: build
	./bin/lazy-dbx

# Quality
lint:
	golangci-lint run ./...

lint-fix:
	golangci-lint run --fix ./...

fmt:
	goimports -w -local github.com/kdjun99/lazy-dbx .
	gofmt -s -w .

vet:
	go vet ./...

# Test
test:
	go test ./...

test-integration:
	go test -tags=integration ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# All checks (run before commit)
check: fmt lint vet test

# Clean
clean:
	rm -rf bin/ coverage.out coverage.html
