# A Self-Documenting Makefile: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html

# Project variables
PACKAGE = github.com/sagikazarmark/go-gin-gorm-opencensus
BINARY_NAME = go-gin-gorm-opencensus

# Build variables
BUILD_DIR = build

# Dependency versions
DEP_VERSION = 0.5.0
GOLANGCI_VERSION = 1.10.2

bin/dep: bin/dep-${DEP_VERSION}
bin/dep-${DEP_VERSION}:
	@mkdir -p bin
	@rm -rf bin/dep-*
	curl https://raw.githubusercontent.com/golang/dep/master/install.sh | INSTALL_DIRECTORY=./bin DEP_RELEASE_TAG=v${DEP_VERSION} sh
	@touch $@

.PHONY: vendor
vendor: bin/dep ## Install dependencies
	bin/dep ensure -v -vendor-only

.PHONY: clean
clean: reset ## Clean the working area and the project
	rm -rf bin/ ${BUILD_DIR}/ vendor/

.PHONY: run
run: build ## Build and execute a binary
	${BUILD_DIR}/${BINARY_NAME} ${ARGS}

.PHONY: build
build: ## Build a binary
	CGO_ENABLED=0 go build -o ${BUILD_DIR}/${BINARY_NAME} ${PACKAGE}

docker-compose.override.yml: ## Create docker compose override file
	cp docker-compose.override.yml.dist docker-compose.override.yml

.PHONY: start
start: docker-compose.override.yml # Start docker development environment
	docker-compose up -d

.PHONY: stop
stop: # Stop docker development environment
	docker-compose stop

.PHONY: reset
reset: # Reset docker development environment
	docker-compose down
	rm -rf .docker/

.PHONY: check
check: test lint ## Run tests and linters

.PHONY: test
test: ## Run all tests
	go test -v ./...

bin/golangci-lint: bin/golangci-lint-${GOLANGCI_VERSION}
bin/golangci-lint-${GOLANGCI_VERSION}:
	@mkdir -p bin
	@rm -rf bin/golangci-lint-*
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | bash -s -- -b ./bin/ v${GOLANGCI_VERSION}
	@touch $@

.PHONY: lint
lint: bin/golangci-lint ## Run linter
	bin/golangci-lint run

.PHONY: help
.DEFAULT_GOAL := help
help:
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

# Variable outputting/exporting rules
var-%: ; @echo $($*)
varexport-%: ; @echo $*=$($*)
