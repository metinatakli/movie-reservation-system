include .env

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
	go run ./cmd/api -db-dsn=${DB_DSN} -redis-url=${REDIS_URL} -smtp-username=${SMTP_USERNAME} -smtp-password=${SMTP_PASSWORD} \
	-stripe-key=${STRIPE_KEY} -stripe-webhook-secret=${STRIPE_WEBHOOK_SECRET}

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

## db/migrations/down: rollback last migration
.PHONY: db/migrations/down
db/migrations/down: confirm
	@echo 'Rolling back last migration...'
	migrate -path ./migrations -database ${DB_DSN} down 1

## db/migrations/reset: rollback all migrations and reapply them
.PHONY: db/migrations/reset
db/migrations/reset: confirm
	@echo 'Resetting database...'
	migrate -path ./migrations -database ${DB_DSN} down
	migrate -path ./migrations -database ${DB_DSN} up

## db/seed file=<mock_file.sql>: Load specific mock data file
.PHONY: db/seed
db/seed:
	@echo 'Seeding database with $(file)...'
ifeq ($(ENV), development)
	docker exec -i ${DB_CONTAINER} psql -U ${POSTGRES_USER} -d ${POSTGRES_DB} < ./migrations/seed/$(file)
else
	@echo 'Skipping seeding (only allowed in development).'
endif

## db/seed/list: List available mock data files
.PHONY: db/seed/list
db/seed/list:
	@ls -1 ./migrations/seed/*.sql

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
	@echo 'Vetting test code...'
	testifylint ./...
	@echo 'Running tests...'
	go test -vet=off ./...

## build: build the application
.PHONY: build
build:
	@echo 'Building cmd/api...'
	go build -ldflags='-s' -o=./bin/api ./cmd/api

## docker/build: build the Docker image
.PHONY: docker/build
docker/build:
	@echo 'Building Docker image...'
	docker-compose build

## docker/up: start the Docker containers
.PHONY: docker/up
docker/up:
	@echo 'Starting Docker containers...'
	docker-compose up -d

## docker/down: stop the Docker containers
.PHONY: docker/down
docker/down:
	@echo 'Stopping Docker containers...'
	docker-compose down