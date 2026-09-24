# SPDX-FileCopyrightText: 2025 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

PROJECT_NAME 	?= credimi
ORGANIZATION 	?= forkbombeu
ROOT_DIR		?= $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
# COMPOSE_PROJECT_NAME is owned by scripts/worktree-env.sh (same sanitization everywhere).
COMPOSE_PROJECT_NAME ?= $(shell "$(ROOT_DIR)/scripts/worktree-env.sh" project-name)
COMPOSE_DEV_OVERRIDE_FILE ?= /tmp/$(COMPOSE_PROJECT_NAME)-docker-compose.dev.yaml
PROCFILE_DEV ?= $(ROOT_DIR)/Procfile.dev
PROCFILE_RUNTIME ?= /tmp/$(COMPOSE_PROJECT_NAME)-Procfile.dev
WORKTREE_ENV_FILE ?= $(ROOT_DIR)/.env.worktree
DEV_PORTS_DEFAULTS ?= $(ROOT_DIR)/scripts/dev-ports.env
DEV_COMPOSE_ENV ?= $(ROOT_DIR)/scripts/dev-compose.env
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
COVERAGE_FILE	?= $(COVOUT)
COVERAGE_MIN	?= 80
COVERAGE_PKGS ?= $(shell go list ./... | grep -v '/webapp/node_modules/')

# Submodules
WEBENV			= $(WEBAPP)/.env
BIN				= $(ROOT_DIR)/.bin
DEPS 			= mise git temporal wget
TEST_DEPS		= mise

# Generic tool checker
define require_tools
	@missing=0; \
	for t in $(1); do \
		if ! command -v $$t >/dev/null 2>&1; then \
			echo "🧧 \`$$t\` not found in PATH, please install it." >&2; \
			missing=1; \
		fi; \
	done; \
	if [ $$missing -ne 0 ]; then \
		echo "🥶 Missing required tools, aborting."; \
		exit 1; \
	fi
endef

# scripts/dev-ports.env is the single source of truth for classic ports.
# Optional .env.worktree overrides them for parallel worktrees.
# scripts/dev-compose.env pins Compose image versions.
# Runtime Compose/Procfile files are written by scripts/worktree-dev-prepare.sh
# (via scripts/worktree-compose.sh for make dev / worktree-down / purge).

all: help
.PHONY: submodules version dev dev.noworkers worktree-bootstrap worktree-down worktree-destroy test test.all lint tidy purge build docker docker-tunnel doc clean tools help w devtools coverage-check fcaf-run fcaf-sync

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
	$(call require_tools,$(DEPS) $(DEV_DEPS))
	@bash -c 'set -euo pipefail; \
		_user_internal="$${CREDIMI_INTERNAL_APP_URL-}"; \
		eval "$$(./scripts/worktree-env.sh export)"; \
		if [ -n "$${_user_internal}" ]; then export CREDIMI_INTERNAL_APP_URL="$${_user_internal}"; fi; \
		unset PORT; \
		export PUBLIC_TURNSTILE_SITE_KEY="$${PUBLIC_TURNSTILE_SITE_KEY:-1x00000000000000000000AA}"; \
		export TURNSTILE_SECRET_KEY="$${TURNSTILE_SECRET_KEY:-1x0000000000000000000000000000000AA}"; \
		./scripts/worktree-compose.sh prepare; \
		trap "./scripts/worktree-compose.sh stop" EXIT; \
		./scripts/worktree-compose.sh up; \
		DEBUG=1 $(GOTOOL) hivemind -T -l API,UI -d "$(ROOT_DIR)" "$${PROCFILE_RUNTIME}"'

dev.noworkers: ## 🚀 run in watch mode without Temporal workers
	CREDIMI_TEMPORAL_WORKERS_DISABLED=1 $(MAKE) dev

worktree-bootstrap: ## 🌳 Worktrunk bootstrap: copy-ignored + .env.worktree ports
	bash ./scripts/worktree-bootstrap.sh

worktree-down: ## 🌳 stop this worktree Compose project
	@./scripts/worktree-compose.sh down

worktree-destroy: ## 🗑️ remove a disposable worktree, its runtime data, and branch (WORKTREE=<branch>)
	@if [ -z "$(WORKTREE)" ]; then \
		echo "usage: make worktree-destroy WORKTREE=<branch>" >&2; \
		exit 2; \
	fi
	@if [ "$(WORKTREE)" = "$(GIT_BRANCH)" ]; then \
		echo "refusing to destroy the current worktree branch: $(GIT_BRANCH)" >&2; \
		exit 2; \
	fi
	@command -v wt >/dev/null 2>&1 || { echo "Worktrunk (wt) is required" >&2; exit 1; }
	@wt remove --yes --foreground --reap --force --force-delete "$(WORKTREE)"

test: ## 🧪 run tests
	$(call require_tools,$(TEST_DEPS))
	bash ./scripts/test-summary.sh

test.all: ## 🧪 run all tests, including long tests skipped by test
	$(call require_tools,$(TEST_DEPS))
	TEST_SHORT=0 bash ./scripts/test-summary.sh

testdata.refresh: ## 🗄️ Apply pending migrations into test_pb_data (run after PocketBase upgrades)
	$(GOCMD) run ./cmd/testdata-refresh
fcaf-generate: ## Generate one complete FCAF validation pipeline from scenario sources
	$(GOCMD) run ./cmd/fcaf-pipeline-gen

fcaf-run: fcaf-generate ## Run complete FCAF validation through Credimi and store results server-side
	$(GOCMD) run . fcaf run $(if $(API_KEY),--api-key "$(API_KEY)",) --instance "$(or $(INSTANCE),http://localhost:8090)" --dir "$(or $(FCAF_DIR),config_templates/fcaf/wallet_solution/relying_party/pipelines)" $(if $(FCAF_OUTPUT),--output "$(FCAF_OUTPUT)",) $(if $(FCAF_FILTER),--filter "$(FCAF_FILTER)",) $(if $(FCAF_DEVICE_ID),--device-id "$(FCAF_DEVICE_ID)",)

fcaf-sync: fcaf-generate ## Synchronize complete FCAF pipeline and credential definitions without running them
	$(GOCMD) run . fcaf sync $(if $(API_KEY),--api-key "$(API_KEY)",) --instance "$(or $(INSTANCE),http://localhost:8090)" --dir "$(or $(FCAF_DIR),config_templates/fcaf/wallet_solution/relying_party/pipelines)" --imports-dir "$(or $(FCAF_IMPORTS_DIR),config_templates/fcaf/imports)" $(if $(FCAF_DEVICE_ID),--device-id "$(FCAF_DEVICE_ID)",)
ifeq (test.p, $(firstword $(MAKECMDGOALS)))
  test_name := $(wordlist 2, $(words $(MAKECMDGOALS)), $(MAKECMDGOALS))
  $(eval $(test_name):;@true)
endif
test.p: tools ## 🍷 watch tests and run on change for a certain folder
	$(GOTOOL) gow test -run "^$(test_name)$$" $(GODIRS)

coverage: devtools # ☂️ run test and open code coverage report
	$(GOTEST) -tags=unit -covermode=atomic -coverprofile=$(COVOUT) ./...
	$(GOTOOL) cover -html=$(COVOUT) -o coverage.html
	$(GOTOOL) go-cover-treemap -coverprofile $(COVOUT) > coverage.svg && open coverage.svg

coverage-check: ## ☂️ run tests and fail if total coverage is below COVERAGE_MIN
	@$(GOTEST) -tags=unit -covermode=atomic -coverprofile=$(COVERAGE_FILE) $(COVERAGE_PKGS)
	@awk -v min="$(COVERAGE_MIN)" -v file="$(COVERAGE_FILE)" '\
		BEGIN { covered = 0; total = 0 } \
		NR == 1 && $$1 == "mode:" { next } \
		NF >= 3 { \
			n = $$2 + 0; \
			c = $$3 + 0; \
			total += n; \
			if (c > 0) covered += n; \
		} \
		END { \
			if (total == 0) { \
				printf("Failed to compute coverage from %s: no statement data found\n", file) > "/dev/stderr"; \
				exit 1; \
			} \
			cov = (covered * 100) / total; \
			printf("Total coverage: %.1f%% (required: %s%%)\n", cov, min); \
			if ((cov + 0) < (min + 0)) exit 1; \
		} \
	' $(COVERAGE_FILE)

lint: devtools ## 📑 lint rules checks
	$(call require_tools,$(TEST_DEPS))
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
	@./scripts/worktree-compose.sh down-v
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
	@bash -euc 'set -a; . "$(DEV_COMPOSE_ENV)"; set +a; \
	if [ -z "$${CREDIMI_EXTRA_PAT-}" ] && command -v gh >/dev/null 2>&1; then CREDIMI_EXTRA_PAT="$$(gh auth token 2>/dev/null || true)"; fi; \
	EXTRA_BUILD_ARGS=""; [ -n "$${CREDIMI_EXTRA_PAT-}" ] && EXTRA_BUILD_ARGS="--build-arg CREDIMI_EXTRA_PAT=$$CREDIMI_EXTRA_PAT"; \
	export COMPOSE_PROJECT_NAME="$(COMPOSE_PROJECT_NAME)" CREDIMI_ELASTIC_PASSWORD="$${CREDIMI_ELASTIC_PASSWORD:-devpassword}"; \
	docker compose build --build-arg PUBLIC_POCKETBASE_URL="http://localhost:8090" --build-arg PUBLIC_TURNSTILE_SITE_KEY="$${PUBLIC_TURNSTILE_SITE_KEY:?PUBLIC_TURNSTILE_SITE_KEY is required}" $$EXTRA_BUILD_ARGS; \
	docker compose up'

docker-tunnel: $(DATA) submodules ## 🌐 run docker (detached, logs hidden) and expose http://localhost:8090 over a public cloudflared tunnel
	$(call require_tools,cloudflared)
	@bash -euc 'set -a; . "$(DEV_COMPOSE_ENV)"; set +a; \
	if [ -z "$${CREDIMI_EXTRA_PAT-}" ] && command -v gh >/dev/null 2>&1; then CREDIMI_EXTRA_PAT="$$(gh auth token 2>/dev/null || true)"; fi; \
	EXTRA_BUILD_ARGS=""; [ -n "$${CREDIMI_EXTRA_PAT-}" ] && EXTRA_BUILD_ARGS="--build-arg CREDIMI_EXTRA_PAT=$$CREDIMI_EXTRA_PAT"; \
	export COMPOSE_PROJECT_NAME="$(COMPOSE_PROJECT_NAME)" CREDIMI_ELASTIC_PASSWORD="$${CREDIMI_ELASTIC_PASSWORD:-devpassword}"; \
	docker compose build --build-arg PUBLIC_POCKETBASE_URL="" --build-arg PUBLIC_TURNSTILE_SITE_KEY="$${PUBLIC_TURNSTILE_SITE_KEY:?PUBLIC_TURNSTILE_SITE_KEY is required}" $$EXTRA_BUILD_ARGS; \
	printf "$(CYAN)🐳 Starting docker compose detached (runtime logs hidden — run \`docker compose logs -f\` to view)...$(RESET)\n"; \
	docker compose up -d --remove-orphans; \
	trap "printf \"\n$(YELLOW)🛑 Tunnel closed; stopping containers...$(RESET)\n\"; docker compose down" EXIT INT TERM; \
		printf "$(CYAN)⏳ Waiting for http://localhost:8090 ...$(RESET)\n"; \
		./scripts/wait-for-it.sh localhost:8090 --timeout=180 --quiet || printf "$(YELLOW)⚠️  Service not reachable yet, starting tunnel anyway$(RESET)\n"; \
		printf "\n$(GREEN)🌍 Public URL will appear in the cloudflared banner below:$(RESET)\n\n"; \
		cloudflared tunnel --url http://localhost:8090 \
	'

waf-emulator: ## 🛡  emulate a Cloudflare-proxied deployment (challenges non-browser callers) on :8091
	$(GOCMD) run ./scripts/waf-emulator

## Misc

doc: ## 📚 Serve documentation on localhost with --host
	cd $(DOCS) && bun install
	cd $(DOCS) && bun run docs:dev --open --host

clean: ## 🧹 Clean files and caches
	@$(GOCLEAN)
	@rm -f $(BINARY_NAME)
	@rm -f $(BINARY_NAME)-ui
	@rm -fr $(WEBAPP)/build
	@rm -fr $(WEBAPP)/node_modules
	@rm -fr $(WEBAPP)/.svelte-kit
	@rm -f $(DOCS)/.astro/*
	@rm -f $(COVOUT) coverage.html coverage.svg
	@echo "🧹 cleaned"

generate: $(ROOT_DIR)/pkg/gen.go
	$(GOGEN) $(ROOT_DIR)/pkg/gen.go

# Go Client-column emit + bun CollectionModel / PB type injectors.
# Requires webapp/pb_data/data.db for pocketbase-typegen (same as generate:types).
# Does not alter plain `make generate` (Go-only) or webapp predev.
generate-catalog-wire: ## Regenerate catalog FE wire from schema.go through bun injectors
	$(GOGEN) ./pkg/conformancecatalog
	cd $(WEBAPP) && bun run generate:collections-models && bun run generate:types

devtools: generate

tools: generate $(BIN)
	mise install -y
	ln -sf "$$(mise which et-tu-cesr)" "$(BIN)/et-tu-cesr"
	ln -sf "$$(mise which stepci-captured-runner)" "$(BIN)/stepci-captured-runner"
	test -x "$(BIN)/et-tu-cesr" && test -x "$(BIN)/stepci-captured-runner"

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

kill-pocketbase: ## 🔪 Kill any running PocketBase instance for this worktree API_PORT
	@bash -c 'set -euo pipefail; \
		eval "$$(./scripts/worktree-env.sh export)"; \
		echo "Killing PocketBase on :$${API_PORT}..."; \
		lsof -ti:"$${API_PORT}" -sTCP:LISTEN | xargs kill -9 2>/dev/null || true'
seed: ## 🌱 Seed the database
	@$(GOCMD) run main.go migrate up && $(GOCMD) run cmd/seeds/seed.go 
