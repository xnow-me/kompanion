ifneq (,$(wildcard ./.env))
    include .env
    export
else
	include .env.example
	export
endif

LOCAL_BIN:=$(CURDIR)/bin
PATH:=$(LOCAL_BIN):$(PATH)

COMPOSE_CMD := $(shell command -v docker-compose >/dev/null 2>&1 && echo "docker-compose" || echo "docker compose")

# HELP =================================================================================================================
# This will output the help for each task
# thanks to https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
.PHONY: help

help: ## Display this help screen
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

compose-up: ### Run docker-compose
	$(COMPOSE_CMD) up --build -d postgres && $(COMPOSE_CMD) logs -f
.PHONY: compose-up

compose-all: ### Run all containers
	UID=$(id -u) GID=$(id -g) $(COMPOSE_CMD) up --build
.PHONY: compose-all

compose-up-integration-test: ### Run docker-compose with integration test
	$(COMPOSE_CMD) -f docker-compose.yml -f docker-compose-integration.yml up --build --abort-on-container-exit --exit-code-from integration
.PHONY: compose-up-integration-test

compose-down: ### Down docker-compose
	$(COMPOSE_CMD) down --remove-orphans
.PHONY: compose-down

run: ### swag run
	go mod tidy && go mod download && \
	GIN_MODE=debug go run -tags migrate ./cmd/app
.PHONY: run

docker-rm-volume: ### remove docker volume
	docker volume rm kompanion_pg-data
.PHONY: docker-rm-volume

destroy: compose-down docker-rm-volume ### destroy all
.PHONY: destroy

linter-golangci: ### check by golangci linter
	golangci-lint run
.PHONY: linter-golangci

linter-hadolint: ### check by hadolint linter
	git ls-files --exclude='Dockerfile*' --ignored | xargs hadolint
.PHONY: linter-hadolint

linter-dotenv: ### check by dotenv linter
	dotenv-linter
.PHONY: linter-dotenv

test: ### run test
	go test -v -cover ./internal/...
.PHONY: test

integration-test: ### run integration-test
	go clean -testcache && go test -v ./integration-test/...
.PHONY: integration-test

mock: ### run mockgen
	mockgen -source ./internal/usecase/interfaces.go -package usecase_test > ./internal/usecase/mocks_test.go
.PHONY: mock

migrate-create:  ### create new migration
	migrate create -ext sql -dir migrations '$(MIGRATE_NAME)'
.PHONY: migrate-create

migrate-up: ### migration up
	migrate -path migrations -database '$(KOMPANION_PG_URL)?sslmode=disable' up
.PHONY: migrate-up

bin-deps:
	GOBIN=$(LOCAL_BIN) go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	GOBIN=$(LOCAL_BIN) go install github.com/golang/mock/mockgen@latest
