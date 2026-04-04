VERSION ?= dev
BINARY  := sweo
GOFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build test clean build-frontend dev

# Build Go binary (uses whatever frontend is in frontend_dist/)
build:
	go build $(GOFLAGS) -o $(BINARY) ./cmd/sweo/

# Build frontend and copy to embed directory, then build Go binary
build-all: build-frontend build

# Build frontend assets
build-frontend:
	cd frontend && bun install && bun run build
	rm -rf internal/server/frontend_dist
	cp -r frontend/dist internal/server/frontend_dist

# Run tests
test:
	go test ./... -count=1

# Clean build artifacts
clean:
	rm -f $(BINARY) $(BINARY)-linux-* $(BINARY)-darwin-*
	rm -rf internal/server/frontend_dist
	mkdir -p internal/server/frontend_dist
	touch internal/server/frontend_dist/.gitkeep

# Development: run Go server with hot reload (frontend dev server runs separately)
dev:
	go run ./cmd/sweo/ start

# Cross-compile
build-linux: build-frontend
	GOOS=linux GOARCH=amd64 go build $(GOFLAGS) -o $(BINARY)-linux-amd64 ./cmd/sweo/

build-darwin: build-frontend
	GOOS=darwin GOARCH=arm64 go build $(GOFLAGS) -o $(BINARY)-darwin-arm64 ./cmd/sweo/
