.PHONY: web build run demo dev test check

web:
	cd web && npm ci --no-audit --no-fund && npm run build

build: web
	CGO_ENABLED=0 go build -trimpath -o bin/status ./cmd/status

# Serve the built UI and API on http://localhost:3000, reading Gatus.
run: build
	./bin/status

# Same, with a made-up history and no Gatus. STATUS_DEMO=outage shows trouble.
demo: build
	STATUS_DEMO=true ./bin/status

# API on :3000 with made-up data plus the Vite dev server (hot reload) on :5173.
dev:
	@test -d web/dist || (cd web && npm ci --no-audit --no-fund && npm run build)
	STATUS_DEMO=true go run ./cmd/status & cd web && npm run dev

test:
	go test -race ./...

check: test
	cd web && npm run typecheck
	go vet ./...
