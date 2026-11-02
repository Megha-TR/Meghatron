.PHONY: build test coverage check web-check etcd-up etcd-down demo e2e
build:
	go build -o bin/orchestration-server ./cmd/orchestration-server
	go build -o bin/node-agent ./cmd/node-agent
	go build -o bin/orc ./cmd/cli
test:
	go test -race ./cmd/... ./pkg/... ./internal/...
coverage:
	go test -coverprofile=coverage.out ./pkg/agent ./pkg/api ./pkg/controller ./pkg/resources ./pkg/runtime ./pkg/scheduler
check:
	@test -z "$$(gofmt -l cmd pkg internal)" || (echo 'Run gofmt -w cmd pkg internal'; exit 1)
	go vet ./cmd/... ./pkg/... ./internal/...
	go test -race ./cmd/... ./pkg/... ./internal/...
web-check:
	cd web && npm run typecheck && npm run lint && npm run build
etcd-up:
	docker compose up -d --wait
etcd-down:
	docker compose down
demo:
	bash examples/deploy.sh
e2e:
	bash scripts/test-e2e.sh
