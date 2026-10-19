-include .env
export

COMPOSE := docker compose --project-directory . -f infra/docker/docker-compose.yaml
GRPC_ADDR ?= localhost:50051
GRPC_PROTOS := -proto financecore/v1/categories.proto -proto financecore/v1/expenses.proto -proto financecore/v1/user_telegram_contexts.proto

.PHONY: run build test test-integration vet check tidy lint \
	docker-build docker-up docker-down docker-logs \
	migrate-up migrate-down migrate-force migrate-create migrate-version \
	sqlc-generate proto-generate proto-lint \
	grpc-list grpc-describe \
	gql-init gql-generate

run:
	go run ./cmd

build:
	go build -o bin/financecore ./cmd

test:
	go test -count=1 ./test/...

test-integration:
	go test -race -count=1 -tags=integration ./test/...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

check: vet lint test proto-lint

tidy:
	go mod tidy


## --- local infra (Postgres via docker-compose) ---
# make docker-up          -> all services (db, migration, core)
# make docker-up svc=db   -> only the listed service(s), e.g. svc="db migration"
# make docker-build svc=migration VERSION=0.0.1 -> build migration service with tag 0.0.1
# make docker-down args="-v --remove-orphans" -> adding arguments to deletion

docker-build: svc ?= core
docker-build:
	docker build \
		-f infra/docker/$(if $(filter migration,$(svc)),migration.Dockerfile,Dockerfile) \
		-t revirathya/revirathya-finance-$(svc):$(if $(filter migration,$(svc)),$(MIGRATION_VERSION),$(VERSION)) .

docker-up:
	$(COMPOSE) up -d $(svc)

docker-down:
	$(COMPOSE) down $(args)

docker-logs:
	$(COMPOSE) logs -f $(svc)

## --- migrations (golang-migrate) ---

migrate-up:
	migrate -path migrations -database "$(DATABASE_URI)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URI)" down 1

migrate-force:
	migrate -path migrations -database "$(DATABASE_URI)" force $(version)

migrate-create:
	migrate create -ext sql -dir migrations -seq $(name)

migrate-version:
	migrate -path migrations -database "$(DATABASE_URI)" version

## --- codegen ---

sqlc-generate:
	sqlc generate

proto-generate:
	buf generate

proto-lint:
	buf lint

gql-init:
	go run github.com/99designs/gqlgen init

gql-generate:
	go run github.com/99designs/gqlgen generate

## --- gRPC dev tools (reflection isn't registered, so proto files are passed explicitly) ---

grpc-list:
	grpcurl -plaintext -import-path proto $(GRPC_PROTOS) $(GRPC_ADDR) list

grpc-describe:
	grpcurl -plaintext -import-path proto $(GRPC_PROTOS) $(GRPC_ADDR) describe $(symbol)
