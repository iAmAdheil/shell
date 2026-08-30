.PHONY: dev-web dev-server start build-web build-server db-up db-down test test-server

dev-web:
	cd web && npm run dev

dev-server:
	cd server && go run ./cmd/server

start:
	$(MAKE) -j2 dev-web dev-server

build-web:
	cd web && npm run build

build-server:
	cd server && go build -o bin/server ./cmd/server

db-up:
	docker compose up -d --wait

db-down:
	docker compose down

test: test-server

test-server:
	cd server && go test ./...
