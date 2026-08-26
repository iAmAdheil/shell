.PHONY: dev-web dev-server start build-web build-server

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
