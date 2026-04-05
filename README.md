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
| UI | Tailwind CSS + lucide-react + uPlot + dnd-kit |
| Deployment | NixOS systemd services |
| Remote access | Tailscale |

---

## Requirements

- Neptune Apex running AOS 5+, accessible on the local network
- NixOS (primary target) or any Linux system with Go 1.25+
- Node.js 22+ for frontend builds
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

### 3. Start the backend

> **Note:** If the production systemd service is already running on this machine (port 8420), set a different port in `.env` to avoid conflicts:
> ```bash
> SYMBIONT_API_PORT=8421
> ```

```bash
set -a && source .env && set +a
go run ./cmd/symbiont serve
```

This starts the API server and poller together in a single process. On first start, a token is printed to stdout — save it, it's shown once.

Verify the API is working:

```bash
curl -s -H "Authorization: Bearer <your-token>" http://localhost:8421/api/probes | jq .
```

### 4. Start the frontend dev server

In a second terminal:

```bash
cd frontend
npm install
VITE_API_URL=http://localhost:8421 npm run dev
```

If running on the default port 8420 (no prod service conflict), omit `VITE_API_URL`.

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

# Outlet states and control
symbiont outlets list
symbiont outlets set <outlet-id> OFF
symbiont outlets events

# Alert rules
symbiont alerts list
symbiont alerts create
symbiont alerts update <id>
symbiont alerts delete <id>
symbiont alerts events

# Notification channels
symbiont notify list
symbiont notify create
symbiont notify delete <id>
symbiont notify test <id>

# Display config (probe/outlet labels and ordering)
symbiont config probes list
symbiont config probes update <name>
symbiont config outlets list
symbiont config outlets update <id>

# System
symbiont system status
symbiont system backup
symbiont system cleanup
symbiont system backups
symbiont system log

# Token management
symbiont auth tokens list
symbiont auth tokens create --label "claude-desktop"
symbiont auth tokens revoke <id>
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

Available tools: `get_current_parameters`, `get_probe_history`, `get_outlet_states`, `control_outlet`, `get_outlet_event_log`, `get_alert_rules`, `get_alert_events`, `get_system_status`, `get_system_log`, `get_devices`, `summarize_tank_health`.

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

Currently in **Phase 7: Layout Builder** (phases 1–6 substantially complete).

See `docs/impl-07-layout-builder.md` for the active task list.

---

## License

Private project. Not open source.
