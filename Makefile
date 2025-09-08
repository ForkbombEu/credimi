# SPDX-FileCopyrightText: 2025 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

PROJECT_NAME 	?= credimi
ORGANIZATION 	?= forkbombeu
ROOT_DIR		?= $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
BINARY_NAME 	?= $(PROJECT_NAME)
CLI_NAME		?= $(PROJECT_NAME)-cli
SUBDIRS			?= ./...
MAIN_SRC 		?= $(ROOT_DIR)/main.go
CLI_SRC			?= $(ROOT_DIR)/cli/main.go
DATA			?= $(ROOT_DIR)/pb_data
WEBAPP			?= $(ROOT_DIR)/webapp
GO_SRC 			:= $(wildcard **/*.go)
GODIRS			:= ./pkg/... ./cmd/...
UI_SRC			:= $(shell find $(WEBAPP)/src -type f \( -name '*.svelte' -o -name '*.js' -o -name '*.ts' -o -name '*.css' \) ! -name '*.generated.ts' ! -path 'webapp/src/modules/i18n/paraglide/*')
DOCS			?= $(ROOT_DIR)/docs

GOCMD 			?= go
GOBUILD			?= $(GOCMD) build
GOCLEAN			?= $(GOCMD) clean
GOTEST			?= $(GOCMD) test
GOTOOL			?= $(GOCMD) tool
GOGET			?= $(GOCMD) get
GOFMT			?= $(GOCMD) fmt
GOMOD			?= $(GOCMD) mod
GOINST			?= $(GOCMD) install
GOGEN			?= $(GOCMD) generate
GOPATH 			?= $(shell $(GOCMD) env GOPATH)
GOBIN 			?= $(GOPATH)/bin
GOMOD_FILES 	:= go.mod go.sum
COVOUT			:= coverage.out

# Submodules
WEBENV			= $(WEBAPP)/.env
BIN				= $(ROOT_DIR)/.bin
DEPS 			= mise git temporal wget
DEV_DEPS		= pre-commit
K 				:= $(foreach exec,$(DEPS), $(if $(shell which $(exec)),some string,$(error "🥶 `$(exec)` not found in PATH please install it")))

all: help
.PHONY: submodules version dev test lint tidy purge build docker doc clean tools help w devtools

$(BIN):
	@mkdir -p $@

submodules:
	git submodule update --recursive --init

## Hacking
GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
WHITE  := $(shell tput -Txterm setaf 7)
CYAN   := $(shell tput -Txterm setaf 6)
RESET  := $(shell tput -Txterm sgr0)

VERSION_STRATEGY 	= semver # git, semver, date
VERSION 			:= $(shell cat VERSION 2>/dev/null || echo "0.1.0")
GIT_COMMIT 			?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_BRANCH 			?= $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
BUILD_TIME 			?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')
BUILD_BY 			?= $(shell whoami)

version: ## ℹ️ Display version information
	@echo "$(CYAN)Version:	$(RESET)$(VERSION)"
	@echo "$(CYAN)Commit:		$(RESET)$(GIT_COMMIT)"
	@echo "$(CYAN)Branch:		$(RESET)$(GIT_BRANCH)"
	@echo "$(CYAN)Built:		$(RESET)$(BUILD_TIME)"
	@echo "$(CYAN)Built by: 	$(RESET)$(BUILD_BY)"
	@echo "$(CYAN)Go version:	$(RESET)$(shell $(GOCMD) version)"

$(WEBENV):
	cp $(WEBAPP)/.env.example $(WEBAPP)/.env

$(DATA):
	mkdir -p $(DATA)

dev: $(WEBENV) tools devtools submodules $(BIN) $(DATA) ## 🚀 run in watch mode
	DEBUG=1 $(GOTOOL) hivemind -T Procfile.dev

test: ## 🧪 run tests with coverage
	$(GOTEST) $(GODIRS) -v -race -buildvcs --tags=unit
ifeq (test.p, $(firstword $(MAKECMDGOALS)))
  test_name := $(wordlist 2, $(words $(MAKECMDGOALS)), $(MAKECMDGOALS))
  $(eval $(test_name):;@true)
endif
test.p: tools ## 🍷 watch tests and run on change for a certain folder
	$(GOTOOL) gow test -run "^$(test_name)$$" $(GODIRS)

coverage: devtools # ☂️ run test and open code coverage report
	-$(GOTEST) $(GODIRS) -coverprofile=$(COVOUT)
	$(GOTOOL) cover -html=$(COVOUT) --tags=unit
	$(GOTOOL) go-cover-treemap -coverprofile $(COVOUT) > coverage.svg && open coverage.svg

lint: devtools ## 📑 lint rules checks
	$(GOMOD) tidy -diff
	$(GOMOD) verify
	$(GOCMD) vet $(SUBDIRS)
	$(GOTOOL) govulncheck $(SUBDIRS)
	$(GOTOOL) golangci-lint run $(SUBDIRS)

fmt: devtools ## 🗿 format rules checks
	$(GOFMT) $(GODIRS)

tidy: $(GOMOD_FILES)
	@$(GOMOD) tidy

purge: ## ⛔ Purge the database
	@echo "⛔ Purge the database"
	@rm -rf $(DATA)
	@mkdir $(DATA)

## Deployment

$(BINARY_NAME): $(GO_SRC) tools tidy submodules $(WEBENV)
	@$(GOBUILD) -o $(BINARY_NAME) $(MAIN_SRC)

$(WEBAPP)/build: $(UI_SRC)
	@./$(BINARY_NAME) serve & \
	PID=$$!; \
	./scripts/wait-for-it.sh localhost:8090 --timeout=60; \
	cd $(WEBAPP) && bun i && bun run build; \
	kill $$PID;

$(BINARY_NAME)-ui: $(UI_SRC)
	@./$(BINARY_NAME) serve & \
	PID=$$!; \
	./scripts/wait-for-it.sh localhost:8090 --timeout=60; \
	cd $(WEBAPP) && bun i && bun run bin; \
	kill $$PID;

docker: $(DATA) submodules ## 🐳 run docker with all the infrastructure services
	docker compose build --build-arg PUBLIC_POCKETBASE_URL="http://localhost:8090"
	docker compose up

## Misc

doc: ## 📚 Serve documentation on localhost with --host
	cd $(DOCS) && bun i
	cd $(DOCS) && bun run docs:dev --open --host

clean: ## 🧹 Clean files and caches
	@$(GOCLEAN)
	@rm -f $(BINARY_NAME)
	@rm -f $(BINARY_NAME)-ui
	@rm -fr $(WEBAPP)/build
	@rm -fr $(WEBAPP)/node_modules
	@rm -fr $(WEBAPP)/.svelte-kit
	@rm -f $(DOCS)/.vitepress/config.ts.timestamp*
	@rm -f $(COVOUT) coverage.svg
	@echo "🧹 cleaned"

generate: $(ROOT_DIR)/pkg/gen.go
	$(GOGEN) $(ROOT_DIR)/pkg/gen.go

devtools: generate
	pre-commit install
	pre-commit autoupdate

tools: generate $(BIN) $(BIN)/stepci-captured-runner $(BIN)/et-tu-cesr $(BIN)/maestro/

$(BIN)/stepci-captured-runner:
	wget https://github.com/ForkbombEu/stepci-captured-runner/releases/latest/download/stepci-captured-runner-$(shell uname)-$(shell uname -m) -O $(BIN)/stepci-captured-runner && chmod +x $(BIN)/stepci-captured-runner

$(BIN)/et-tu-cesr:
	wget https://github.com/ForkbombEu/et-tu-cesr/releases/latest/download/et-tu-cesr-$(shell go env GOOS)-$(shell go env GOARCH) -O $(BIN)/et-tu-cesr && chmod +x $(BIN)/et-tu-cesr

$(BIN)/maestro/:
	@echo "Downloading Maestro installer..."
	@mkdir -p $(BIN)/maestro
	@curl -fsSL "https://get.maestro.mobile.dev" -o $(BIN)/maestro/get-maestro.sh
	@chmod +x $(BIN)/maestro/get-maestro.sh
	@MAESTRO_DIR=$(BIN)/maestro $(BIN)/maestro/get-maestro.sh
	@rm -rf $(BIN)/maestro/tmp

## Help:
help: ## Show this help.
	@echo ''
	@echo 'Usage:'
	@echo '  ${YELLOW}make${RESET} ${GREEN}<target>${RESET}'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} { \
		if (/^[a-zA-Z_-]+:.*?##.*$$/) {printf "    ${YELLOW}%-20s${GREEN}%s${RESET}\n", $$1, $$2} \
		else if (/^## .*$$/) {printf "  ${CYAN}%s${RESET}\n", substr($$1,4)} \
		}' $(MAKEFILE_LIST)

kill-pocketbase: ## 🔪 Kill any running PocketBase instance
	@echo "Killing any existing PocketBase instance..."
	@-lsof -ti:8090 -sTCP:LISTEN | xargs kill -9 2>/dev/null || true


