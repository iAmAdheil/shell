.PHONY: dev-web dev-server start build-web build-server build-session-image db-up db-down test test-server test-e2e

# Every local run started from this Makefile has the dev login on, so that
# testing a shared Session as several Users needs no .env edit. The server
# registers those routes only while this is not empty, and nothing here can
# reach a deployed build.
#
# This value wins over a DEV_LOGIN_SECRET in .env, because the process
# environment is read first. Set the .env one for a deployed build.
#
# To run the way production does, with no dev login at all:
#   make start DEV_LOGIN_SECRET=
DEV_LOGIN_SECRET ?= dev
export DEV_LOGIN_SECRET

dev-web:
	cd web && npm run dev

dev-server:
	cd server && go run ./cmd/server

start:
	@if [ -n "$(DEV_LOGIN_SECRET)" ]; then \
		echo "dev login is on. Sign a browser in as a test User with:"; \
		echo "  http://localhost:8081/api/auth/dev/start?u=ada&k=$(DEV_LOGIN_SECRET)"; \
		echo "  http://localhost:8081/api/auth/dev/start?u=grace&k=$(DEV_LOGIN_SECRET)"; \
		echo "Any name works. Use one normal window and one private window."; \
	fi
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

# Browser tests. They need the stack already up, so run `make start` in
# another terminal first. Both sides read the same DEV_LOGIN_SECRET.
test-e2e:
	cd web && npx playwright test
