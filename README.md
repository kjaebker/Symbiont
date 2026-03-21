# Symbiont

A local-first Neptune Apex dashboard and data platform. Replaces Apex Fusion with a self-hosted stack — no cloud, no Neptune account required after initial setup.

---

## What It Does

- Polls your Neptune Apex controller every 10 seconds over the local network
- Stores all probe readings and outlet states in an embedded DuckDB database
- Serves a REST API and real-time SSE stream
- Provides a React dashboard with live charts, history, and outlet control
- Exposes an MCP server so AI assistants (Claude, etc.) can query and control your tank
- Includes a CLI for scripting and terminal access

---

## Stack

| Layer | Technology |
|---|---|
| Backend | Go — four binaries: poller, api, mcp, cli |
| Time-series DB | DuckDB (embedded, no server) |
| App state DB | SQLite (embedded) |
| Frontend | React + TypeScript + Vite |
| UI | shadcn/ui + Tremor + uPlot |
| Deployment | NixOS systemd services |
| Remote access | Tailscale |

---

## Requirements

- Neptune Apex running AOS 5+, accessible on the local network
- NixOS (primary target) or any Linux system with Go 1.23+
- Node.js 20+ for frontend builds
- Tailscale (optional, for remote access)

---

## Quick Start (Development)

### 1. Enter the dev shell

```bash
nix develop
```

This provides Go, DuckDB CLI, SQLite CLI, Node.js, and development tools.

### 2. Configure environment

```bash
cp .env.example .env
```

Edit `.env` with your Apex IP, username, and password:

```bash
SYMBIONT_APEX_URL=http://192.168.1.100
SYMBIONT_APEX_USER=admin
SYMBIONT_APEX_PASS=your-apex-password
```

### 3. Start the poller

```bash
go run ./cmd/poller
```

Wait 30 seconds, then verify data is collecting:

```bash
duckdb telemetry.db "SELECT probe_name, value, ts FROM probe_readings LIMIT 10;"
```

### 4. Start the API server

In a second terminal:

```bash
go run ./cmd/api
```

On first start, a token is printed to stdout. Save it — it's shown once.

Verify the API is working:

```bash
curl -s -H "Authorization: Bearer <your-token>" http://localhost:8420/api/probes | jq .
```

### 5. Start the frontend dev server

```bash
cd frontend
npm install
npm run dev
```

Open [http://localhost:5173](http://localhost:5173), enter your token, and the dashboard loads.

---

## Running All Services (Production)

On NixOS, all services are managed by systemd and defined in `flake.nix`:

```bash
# Enable and start all services
sudo systemctl enable --now symbiont-poller
sudo systemctl enable --now symbiont-api
sudo systemctl enable --now symbiont-mcp

# Check status
sudo systemctl status symbiont-poller
sudo systemctl status symbiont-api

# View logs
sudo journalctl -u symbiont-poller -f
sudo journalctl -u symbiont-api -f
```

The frontend is built and served statically by the API server:

```bash
cd frontend && npm run build
# Frontend is now served at http://localhost:8420/
```

---

## CLI Usage

```bash
# Current probe values
symbiont probes current

# Probe history
symbiont probes history Temp --interval 5m

# Outlet states
symbiont outlets list

# Control an outlet
symbiont outlets set <outlet-id> OFF

# System status
symbiont system status

# Run a manual backup
symbiont system backup

# Token management
symbiont auth tokens list
symbiont auth tokens create --label "claude-desktop"
```

Add `--json` to any command for machine-readable output.

---

## AI Integration (MCP)

Symbiont exposes an MCP server that lets Claude and other AI assistants query and control your tank.

**Claude Desktop:** Add to `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "symbiont": {
      "command": "/run/current-system/sw/bin/symbiont-mcp",
      "env": {
        "SYMBIONT_API_URL": "http://localhost:8420",
        "SYMBIONT_TOKEN": "your-token-here"
      }
    }
  }
}
```

**Claude Code:**

```bash
claude mcp add symbiont /run/current-system/sw/bin/symbiont-mcp \
  --env SYMBIONT_API_URL=http://localhost:8420 \
  --env SYMBIONT_TOKEN=your-token
```

Available tools: `get_current_parameters`, `get_probe_history`, `get_outlet_states`, `control_outlet`, `get_outlet_event_log`, `get_alert_rules`, `get_system_status`, `summarize_tank_health`.

---

## Authentication

All API endpoints require a Bearer token. The token is generated on first run and printed once to stdout. Additional tokens can be created via:

```bash
symbiont auth tokens create --label "phone"
```

Tokens are stored in SQLite. If the original token is lost:

```bash
symbiont auth reset --db-path /var/lib/symbiont/app.db --yes
```

---

## Data

- **DuckDB** (`telemetry.db`) — all probe readings and outlet states, 10-second granularity
- **SQLite** (`app.db`) — alert rules, display config, tokens, outlet event log, backup records

Default retention: 1 year. Configurable via `SYMBIONT_RETENTION_DAYS`. Cleanup runs weekly via systemd timer.

Backups run nightly to `/var/lib/symbiont/backups/` and are viewable in the Settings page.

---

## Remote Access

Use Tailscale rather than exposing port 8420 to the internet. The API uses HTTP (not HTTPS) — Tailscale encrypts traffic end-to-end.

Restrict access via Tailscale ACLs to only your own devices.

---

## Documentation

| Document | Description |
|---|---|
| `CLAUDE.md` | Agent instructions — read before every coding session |
| `docs/architecture.md` | Full technical architecture |
| `docs/impl-00-overview.md` | Implementation plan index |
| `docs/impl-01-*.md` through `impl-07-*.md` | Per-phase task lists |
| `docs/apex-api-notes.md` | Apex local API findings from DevTools capture |

---

## Project Status

Currently in **Phase 1: Data Collection**.

See `docs/impl-01-data-collection.md` for the active task list.

---

## License

Private project. Not open source.
