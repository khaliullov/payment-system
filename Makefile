GOPATH 		:= $(shell go env GOPATH)
GODEP  		:= $(GOPATH)/bin/dep
GOCILINT	:= $(GOPATH)/bin/golangci-lint
BINARY_NAME := payment-system
HTTP_HOST   := localhost

-include .env

.PHONY: all help run up build lint test install at

all: build

help:           ## Show this help
help:
	@echo "Usage:"
	@fgrep -h "##" $(MAKEFILE_LIST) | fgrep -v fgrep | sed -e 's/\\$$//' | sed -e 's/##//'

run:            ## Run script with arguments. example: `make run -- arg1 arg2`
run: vendor
	go run cmd/payment-system/main.go $(filter-out $@, $(MAKECMDGOALS))

up:             ## Start docker compose
up: | .env docker/postgresql/data
	docker-compose up --remove-orphans

build:          ## Build the binary
build: vendor lint test
	go build -o $(BINARY_NAME) cmd/payment-system/main.go

lint:           ## Run golangci-lint
lint: vendor $(GOCILINT)
	$(GOCILINT) run -E dupl -E gofmt -E golint

test:           ## Run go test -cover
test: vendor
	go test -cover ./cmd/... ./pkg/...

at:				## Run acceptance test (purges all data!)
at: vendor
	cd test && \
		HTTP_HOST=$(HTTP_HOST) HTTP_PORT=$(HTTP_PORT) DB_HOST=$(POSTGRES_HOST) \
		DB_PORT=$(POSTGRES_PORT) DB_NAME=$(POSTGRES_DB) DB_USER=$(POSTGRES_USER) \
		DB_PASSWORD=$(POSTGRES_PASSWORD) \
		go test -v -tags=at

install:        ## Install binary
	cp $(BINARY_NAME) /usr/bin/

.env:
	@echo ".env file was not found, creating with defaults"
	cp .env.dist .env

docker/postgresql/data:
	mkdir -p docker/postgresql/data

$(GODEP):
	cd $(GOPATH) && go get -u github.com/golang/dep/cmd/dep

$(GOCILINT):
	cd $(GOPATH) && go get -u github.com/golangci/golangci-lint/cmd/golangci-lint

Gopkg.toml: | $(GODEP)
	$(GODEP) init

vendor: | Gopkg.toml
	@echo "No vendor dir found. Fetching dependencies now..."
	GOPATH=$(GOPATH) $(GODEP) ensure
