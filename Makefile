include .envrc

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

.PHONY: confirm
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

## run: run the application
.PHONY: run
run:
	go run ./cmd/api -db-dsn=${DB_DSN} -redis-url=${REDIS_URL} -smtp-username=${SMTP_USERNAME} -smtp-password=${SMTP_PASSWORD}

## generate: generate the OpenAPI server code
.PHONY: generate
generate:
	go generate ./api

## db/migrations/new name=$1: create a new database migration
.PHONY: db/migrations/new
db/migrations/new:
	@echo 'Creating migration files for ${name}...'
	migrate create -seq -ext=.sql -dir=./migrations ${name}

## db/migrations/up: apply all up database migrations
.PHONY: db/migrations/up
db/migrations/up: confirm
	@echo 'Running up migrations...'
	migrate -path ./migrations -database ${DB_DSN} up

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