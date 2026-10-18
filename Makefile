# docs/engineering/local-development.md section 4, every target runs from the repository root
SHELL := /bin/sh
COMPOSE := docker compose -f deploy/compose.yaml -f deploy/compose.dev.yaml --env-file deploy/env/compose.env
SQLC := docker run --rm -v "$(CURDIR)/api:/src" -w /src sqlc/sqlc:1.31.1
s ?= api

.PHONY: help env dev up down reset logs ps migrate migrate-down migrate-status seed seed-demo admin gen gen-api gen-web \
	check check-api check-web check-ml check-data test-api test-api-integration test-web test-ml test-data eval eval-latency e2e

help:
	@echo "env dev down reset logs s=<service> ps migrate migrate-down seed seed-demo admin email=<x> name=<y>"
	@echo "gen check test-api test-api-integration test-web test-ml test-data e2e"

env:
	@sh deploy/scripts/make-env.sh

dev: env
	$(COMPOSE) up -d --build --wait

up:
	$(COMPOSE) up -d --wait

down:
	$(COMPOSE) down

reset:
	$(COMPOSE) down -v
	$(MAKE) dev

logs:
	$(COMPOSE) logs -f $(s)

ps:
	$(COMPOSE) ps

migrate:
	$(COMPOSE) run --rm migrate migrate up

migrate-down:
	$(COMPOSE) run --rm migrate migrate down

migrate-status:
	$(COMPOSE) run --rm migrate migrate status

seed:
	$(COMPOSE) run --rm seed seed --overlay /app/dev-seed

seed-demo:
	$(COMPOSE) run --rm seed seed --overlay /app/dev-seed --demo

admin:
	$(COMPOSE) run --rm api admin create --email "$(email)" --name "$(name)"

gen: gen-api gen-web

gen-api:
	$(SQLC) generate
	cd api && go generate ./...

gen-web:
	cd web && pnpm exec openapi-typescript ../api/openapi.yaml -o src/lib/api/schema.d.ts

check: check-api check-web check-ml check-data

check-api:
	cd api && test -z "$$(gofmt -l .)" && go vet -tags nodynamic ./... && golangci-lint run ./... && go test -tags nodynamic ./...

check-web:
	cd web && pnpm exec prettier --check . && pnpm lint && pnpm typecheck && pnpm test

check-ml:
	cd ml && uv run ruff format --check . && uv run ruff check . && uv run mypy && uv run pytest

check-data:
	python -m unittest discover -s data/scripts -p "test_*.py"

test-api:
	cd api && go test -tags nodynamic ./...

test-api-integration:
	cd api && go test -tags integration,nodynamic ./...

test-web:
	cd web && pnpm test

test-ml:
	cd ml && uv run pytest

test-data: check-data

EVAL_DATA ?= data/eval
EVAL_SPLIT ?= val

# runs the harness of docs/ai/evaluation.md on the fetched dataset, every number in a report comes from here
eval:
	cd ml && uv run python -m eval.run --dataset ../$(EVAL_DATA) --split $(EVAL_SPLIT) --hf-home $${HF_HOME:-$$HOME/.cache/huggingface}

# the CPU latency of every component on this machine (T9, matching-engine section 10)
eval-latency:
	cd ml && uv run python -m eval.latency

# the auth limiters live in api memory, a restart gives the journeys a fresh BR-SEC-01 budget
e2e:
	$(COMPOSE) restart api
	$(COMPOSE) up -d --wait api
	cd web && pnpm e2e
