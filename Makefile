.PHONY: dev-web dev-server start build-web build-server build-session-image db-up db-down test test-server test-e2e

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

# The image every Session runs in. Build it before the server starts a Session.
build-session-image:
	docker build -f server/session.Dockerfile -t shell-session:latest server

db-up:
	docker compose up -d --wait

db-down:
	docker compose down

test: test-server

test-server:
	cd server && go test ./...

# Browser tests. They need the stack already up, and the server started with
# DEV_LOGIN_SECRET set. Run `make start` in another terminal first.
test-e2e:
	cd web && npx playwright test
