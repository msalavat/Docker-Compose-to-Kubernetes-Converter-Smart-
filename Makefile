VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

.PHONY: build test lint coverage install clean release

build:
	go build -ldflags "-X main.version=$(VERSION)" -o bin/kompoze .

test:
	go test ./... -race -count=1

test-integration:
	go test ./... -tags=integration -v

lint:
	golangci-lint run

coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

install:
	go install -ldflags "-X main.version=$(VERSION)" .

clean:
	rm -rf bin/ dist/ coverage.*

release:
	goreleaser release --snapshot --clean
