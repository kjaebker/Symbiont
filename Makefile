.PHONY: build frontend test clean

# Build the symbiont binary with embedded frontend.
# Requires the frontend to be built first (handled automatically).
build: frontend
	go build -tags release -o symbiont ./cmd/symbiont
	@echo "Built: ./symbiont"
	@echo "Run:   cp .env.example .env && ./symbiont serve"

# Build the frontend assets.
frontend:
	cd frontend && npm ci && npm run build

# Run all Go tests.
test:
	go test ./...

# Remove build output.
clean:
	rm -f symbiont
	rm -rf frontend/dist
