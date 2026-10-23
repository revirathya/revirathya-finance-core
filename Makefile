-include .env
export

COMPOSE := docker compose --project-directory . -f infra/docker/docker-compose.yaml
GRPC_ADDR ?= localhost:51051
GRPC_PROTOS := -proto financecore/v1/categories.proto -proto financecore/v1/expenses.proto -proto financecore/v1/user_telegram_contexts.proto
TRIVY_IMAGE := aquasec/trivy@sha256:be1190afcb28352bfddc4ddeb71470835d16462af68d310f9f4bca710961a41e # 0.70.0

.PHONY: run build test test-integration vet check tidy lint \
	docker-build docker-up docker-down docker-logs \
	migrate-up migrate-down migrate-force migrate-create migrate-version \
	sqlc-generate proto-generate proto-lint \
	grpc-list grpc-describe \
	gql-init gql-generate \
	generate hooks \
	image-scan

run:
	go run -ldflags "-X main.version=$(VERSION)" ./cmd

build:
	go build -ldflags "-X main.version=$(VERSION)" -o bin/financecore ./cmd

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
		$(if $(filter migration,$(svc)),,--build-arg VERSION=$(VERSION)) \
		-t revirathya/revirathya-finance-$(svc):$(if $(filter migration,$(svc)),$(MIGRATION_VERSION),$(VERSION)) .

# make docker-up tracing=1 -> also start Jaeger (profile "tracing") and send core's traces to it
docker-up:
	$(if $(tracing),COMPOSE_PROFILES=tracing DOCKER_OTLP_ENDPOINT=http://jaeger:4317) $(COMPOSE) up $(args) -d $(svc)

docker-down:
	COMPOSE_PROFILES='*' $(COMPOSE) down $(args)

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
PROTOSET := bin/financecore.binpb
grpc-list:
	buf build -o $(PROTOSET)
	grpcurl -protoset $(PROTOSET) list

grpc-describe:
	buf build -o $(PROTOSET)
	grpcurl -protoset $(PROTOSET) describe $(symbol)

# -- Global
generate: proto-generate sqlc-generate gql-generate

hooks:
	git config core.hooksPath .githooks

# make image-scan [svc=migration] -> scan the image built by docker-build, with the same rules as CI
image-scan: svc ?= core
image-scan:
	docker run --rm \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v financecore-trivy-cache:/root/.cache \
		-v $(CURDIR)/.trivyignore:/work/.trivyignore:ro \
		$(TRIVY_IMAGE) image --severity HIGH,CRITICAL --ignore-unfixed --exit-code 1 \
		--ignorefile /work/.trivyignore \
		revirathya/revirathya-finance-$(svc):$(if $(filter migration,$(svc)),$(MIGRATION_VERSION),$(VERSION))
