# Deploying Symbiont's Remote MCP for claude.ai

This guide explains how to expose Symbiont's MCP server over HTTPS so it can
be added as a connector in a [claude.ai Project][projects], giving you tool
access from the mobile app and from claude.ai in the browser.

---

## What you're exposing

Symbiont's API server mounts a Streamable-HTTP MCP endpoint at:

```
POST /api/mcp
```

Auth is the same bearer token used by the REST API.  The MCP endpoint requires
a valid token and enforces the same scope rules:

| Scope | Can do |
|-------|--------|
| `read` | Read-only tools (probes, livestock, journal, alerts, etc.) |
| `control` | Read + outlet control, feed mode |
| `admin` | Everything |

For a mobile / claude.ai connector, create a `read` or `control` scoped token
labeled `claude-mobile` (Settings → Tokens → Create Token).

---

## Option A: Tailscale Funnel (recommended)

[Tailscale Funnel][funnel] gives each device a stable public HTTPS URL without
port forwarding, firewall rules, or a reverse proxy.

### 1. Install Tailscale

Follow the [Tailscale install guide][tailscale-install] for your platform (NixOS
module is available in nixpkgs).

### 2. Enable Funnel for the Symbiont port

```bash
tailscale funnel 8420
```

Your Symbiont instance is now reachable at:

```
https://<machine-name>.<tailnet>.ts.net/api/mcp
```

### 3. Get your public URL

```bash
tailscale funnel status
```

### 4. Add to claude.ai

See [Adding to claude.ai](#adding-to-claudeai) below.

---

## Option B: Cloudflare Tunnel

If you prefer Cloudflare, install `cloudflared` and run:

```bash
cloudflared tunnel --url http://localhost:8420
```

This gives you a randomly-assigned `*.trycloudflare.com` URL (ephemeral) or a
stable custom domain if you own one and configure the tunnel via the Cloudflare
dashboard.

For a permanent setup, create a named tunnel:

```bash
cloudflared tunnel create symbiont
cloudflared tunnel route dns symbiont api.yourdomain.com
cloudflared tunnel run symbiont
```

The MCP endpoint becomes:

```
https://api.yourdomain.com/api/mcp
```

---

## Adding to claude.ai

1. In claude.ai, open a **Project** (or create one).
2. Go to **Project settings → Connectors**.
3. Click **Add connector** and select **Custom MCP server**.
4. Enter:
   - **URL**: `https://<your-host>/api/mcp`
   - **Authorization**: `Bearer <your-token>`
5. Save.  Claude will discover Symbiont's tools automatically.

### Paste the agent persona

Symbiont's MCP surface is richer with context.  In the Project's
**Custom instructions** field, paste the output of:

```bash
symbiont agent context
# or via the API:
curl -H "Authorization: Bearer <token>" https://<your-host>/api/agent/context
```

This gives Claude your tank facts, livestock, target parameters, and persona
preferences without needing to call `get_agent_context` on every turn.

You can also do this from the **Settings → Agent** tab in the dashboard:
click **Copy persona for claude.ai** to copy the context to your clipboard.

---

## Security notes

- Create a dedicated `claude-mobile` token (Settings → Tokens) with `control`
  scope. This limits Claude to read + outlet control without token management.
- The MCP endpoint is protected by the same bearer-token auth as the REST API.
  Anyone with your token can call all tools within that token's scope.
- Tailscale Funnel and Cloudflare Tunnel both provide TLS termination — the
  token is not exposed in transit.
- Revoke the token at any time from Settings → Tokens.

---

## Troubleshooting

| Symptom | Fix |
|---------|-----|
| 401 on `/api/mcp` | Token is missing or invalid. Check the `Authorization: Bearer <token>` header. |
| `HTTP MCP transport disabled` in logs | `SYMBIONT_TOKEN` env var not set. The loopback client needs a token to make internal API calls. |
| Tools return errors | Check `/api/healthz` to verify the API server is reachable from its own loopback interface. |

[projects]: https://support.anthropic.com/en/articles/10065695-claude-ai-projects-overview
[funnel]: https://tailscale.com/kb/1223/funnel
[tailscale-install]: https://tailscale.com/download
