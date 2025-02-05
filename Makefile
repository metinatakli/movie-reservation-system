include .envrc

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

## run: run the application
.PHONY: run
run:
	go run ./cmd/api

## generate: generate the OpenAPI server code
.PHONY: generate
generate:
	go generate ./api	

## tidy: format all .go files, and tidy module dependencies
.PHONY: tidy
tidy:
	@echo 'Formatting go files...'
	go fmt ./...
	@echo 'Tidying module dependencies'
	go mod tidy
	@echo 'Verifying module dependencies...'
	go mod verify

## audit: run quality control checks
.PHONY: audit
audit:
	@echo 'Checking module dependencies'
	go mod tidy -diff
	go mod verify
	@echo 'Vetting code...'
	go vet ./...
	staticcheck ./...
	@echo 'Running tests...'
	go test -vet=off ./...

## build: build the application
.PHONY: build
build:
	@echo 'Building cmd/api...'
	go build -ldflags='-s' -o=./bin/api ./cmd/api