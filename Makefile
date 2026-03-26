.PHONY: build frontend test clean release deploy

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

# ── Release flow ──────────────────────────────────────────────────────
# Usage: make release V=0.3.0
#
# 1. Runs all checks (go build, go test, tsc, frontend build)
# 2. Tags and pushes — CI builds the release artifacts
# 3. Waits for CI, then updates flake.nix with the new hash
# 4. Commits and pushes the hash update
#
# After this completes, deploy on NixOS with: make deploy
release:
ifndef V
	$(error Usage: make release V=0.3.0)
endif
	@echo "==> Preflight checks..."
	go build ./...
	go test ./...
	cd frontend && npx tsc --noEmit
	cd frontend && npm run build
	@echo "==> All checks passed."
	@echo ""
	@echo "==> Tagging v$(V)..."
	git tag -a "v$(V)" -m "v$(V)"
	git push origin "v$(V)"
	@echo ""
	@echo "==> Waiting for CI release build..."
	@echo "    Waiting for run to appear..."; \
	for i in 1 2 3 4 5 6 7 8 9 10; do \
		RUN_ID=$$(gh run list --json databaseId,headBranch -q '.[] | select(.headBranch=="v$(V)") | .databaseId' --limit 5 | head -1); \
		[ -n "$$RUN_ID" ] && break; \
		sleep 3; \
	done; \
	if [ -z "$$RUN_ID" ]; then echo "ERROR: CI run not found for v$(V)"; exit 1; fi; \
	echo "    Found run $$RUN_ID"; \
	gh run watch "$$RUN_ID" --exit-status
	@echo ""
	@echo "==> Updating flake.nix with new hash..."
	@HASH=$$(nix-prefetch-url "https://github.com/kjaebker/Symbiont/releases/download/v$(V)/symbiont-linux-amd64.tar.gz" 2>/dev/null | xargs nix hash convert --hash-algo sha256 --to sri); \
	echo "    New hash: $$HASH"; \
	sed -i 's|download/v[^/]*/symbiont-linux-amd64|download/v$(V)/symbiont-linux-amd64|' flake.nix; \
	sed -i 's|version = "[^"]*"; # <── bump|version = "$(V)"; # <── bump|' flake.nix; \
	sed -i "s|hash = \"sha256-[^\"]*\"; # <── update|hash = \"$$HASH\"; # <── update|" flake.nix
	@echo ""
	@echo "==> Committing flake.nix update..."
	git add flake.nix
	git commit -m "Update symbiont-bin to v$(V)"
	git push origin main
	@echo ""
	@echo "==> Release v$(V) complete!"
	@echo "    Deploy on NixOS with: make deploy"

# ── Deploy on NixOS ──────────────────────────────────────────────────
# Run this on the NixOS machine after 'make release'.
deploy:
	cd /etc/nixos && sudo nix flake update symbiont
	sudo nixos-rebuild switch
