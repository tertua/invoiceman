.PHONY: clean critic security lint test build run web.check dev-be dev-fe promote check-flow

APP_NAME = apiserver
BUILD_DIR = $(PWD)/build

clean:
	rm -rf ./build

# Packages are listed explicitly so third-party code under web/node_modules
# is never linted. hugeParam/rangeValCopy are style/perf suggestions whose
# refactors would churn many call sites for no behavior change.
critic:
	gocritic check -enableAll -disable hugeParam,rangeValCopy ./app/... ./pkg/... ./platform/... ./docs/... .

security:
	gosec ./...

lint:
	golangci-lint run ./...

test: clean critic security lint
	go test -v -timeout 30s -coverprofile=cover.out -cover ./...
	go tool cover -func=cover.out

build: test
	CGO_ENABLED=0 go build -ldflags="-w -s" -o $(BUILD_DIR)/$(APP_NAME) main.go

run: swag build
	$(BUILD_DIR)/$(APP_NAME)

# Local dev processes (see AGENTS.md "Local dev run"). Binaries and logs
# live under /tmp/opencode (pre-approved scratch dir, never the repo root).
dev-be:
	go build -o /tmp/opencode/invoiceman . && /tmp/opencode/invoiceman

dev-fe:
	npm --prefix web run dev -- --host 0.0.0.0

web.check:
	npm --prefix web run lint
	npm --prefix web run build
	npm --prefix web run check:bundles:strict

docker.run: docker.network docker.postgres swag docker.fiber docker.redis

docker.network:
	docker network inspect template-network >/dev/null 2>&1 || \
	docker network create -d bridge template-network

docker.fiber.build:
	docker build -t apiserver .

docker.fiber: docker.fiber.build
	docker run --rm -d \
		--name template-fiber \
		--network template-network \
		-p 5000:5000 \
		apiserver

docker.postgres:
	docker run --rm -d \
		--name template-postgres \
		--network template-network \
		-e POSTGRES_USER=postgres \
		-e POSTGRES_PASSWORD=password \
		-e POSTGRES_DB=postgres \
		-v ${HOME}/dev-postgres/data/:/var/lib/postgresql/data \
		-p 5432:5432 \
		postgres

docker.redis:
	docker run --rm -d \
		--name template-redis \
		--network template-network \
		-p 6379:6379 \
		redis

docker.stop: docker.stop.fiber docker.stop.postgres docker.stop.redis

docker.stop.fiber:
	docker stop template-fiber

docker.stop.postgres:
	docker stop template-postgres

docker.stop.redis:
	docker stop template-redis

swag:
	swag init

# Branching: dev is daily work, main is stable only.
# Promote only via fast-forward, never merge-commit or force-push.
promote:
	git fetch origin --prune
	git checkout main
	git merge --ff-only origin/dev
	git push origin main
	git checkout dev

check-flow:
	git fetch origin --prune
	@echo "main..dev count (behind ahead): $$(git rev-list --left-right --count origin/main...origin/dev)"
	@git merge-base --is-ancestor origin/main origin/dev || (echo "FAIL: main is not ancestor of dev (needs rebase/ff, no merge-commit/force-push)"; exit 1)
	@test -z "$$(git log --merges --format=%H origin/main..origin/dev)" || (echo "FAIL: merge commits found in main..dev, keep history linear via rebase"; git log --merges --oneline origin/main..origin/dev; exit 1)
	@echo "OK: main ancestor of dev, no merge commits."
