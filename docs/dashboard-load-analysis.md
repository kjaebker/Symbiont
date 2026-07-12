# Dashboard Load Performance — Analysis & Improvement Plan

## Current Data Flow

When the Dashboard page mounts, TanStack Query fires **9 parallel requests** (all fire simultaneously since none are `enabled: false`):

| # | Endpoint | Database(s) | Queries | Notes |
|---|----------|-------------|---------|-------|
| 1 | `GET /api/dashboard` | SQLite | 1 — `SELECT ... FROM dashboard_items ORDER BY sort_order` | Fast, small table |
| 2 | `GET /api/probes` | DuckDB + SQLite | 1 window function scan on `probe_readings` + 1 `SELECT * FROM probe_config` + N INSERT OR IGNORE for new probes | **Heaviest query** — full table window scan. Also does writes (InitProbeConfig) during a read request. |
| 3 | `GET /api/outlets` | DuckDB + SQLite | 1 window function scan on `outlet_states` + 1 `SELECT * FROM outlet_config` | Window scan similar to probes |
| 4 | `GET /api/devices` | SQLite | 3 — devices list + probe links + outlet links | Already batched (no N+1) |
| 5 | `GET /api/system` | DuckDB + filesystem | 2 — `SELECT ... FROM controller_meta LIMIT 1` + `SELECT MAX(ts) FROM probe_readings` + 2x os.Stat() | Fast queries but two separate round-trips to DuckDB |
| 6 | `GET /api/feed` | **Apex HTTP** | 1 HTTP call to Apex `/rest/status` | **Network bottleneck** — waits on Apex controller response (typically 200-800ms) |
| 7 | `GET /api/livestock` | SQLite | 1 — `SELECT ... FROM livestock ORDER BY name` | Fast, small table |
| 8 | `GET /api/tasks/due` | SQLite | 2 — dosing schedules JOIN + maintenance tasks | Small tables, fast |
| 9 | `GET /api/daily-prompt` | SQLite | 1-3 — journal check + audit events (contextual prompt) | Fast, often returns early (< noon or already answered) |

### Total: ~15+ database queries across DuckDB and SQLite, plus 1 external HTTP call

---

## Root Causes of Slow Load

### 1. Feed status hits the Apex controller on every load
`GET /api/feed` makes a live HTTP request to the Apex `/rest/status` endpoint. This is an external network round-trip (200-800ms typical) that blocks the entire dashboard render since React waits for all queries to settle before showing content (`isLoading = layoutLoading || probesLoading || outletsLoading || devicesLoading`).

### 2. DuckDB window function scans on every request
Both `CurrentProbeReadings` and `CurrentOutletStates` run:
```sql
SELECT *, ROW_NUMBER() OVER (PARTITION BY probe_did ORDER BY ts DESC) AS rn
FROM probe_readings
WHERE rn = 1
```
This is a full-table window function scan. With months of polling data (every 10 seconds), these tables can have millions of rows. DuckDB is fast but this still takes measurable time, especially on the NixOS mini PC hardware.

### 3. SSE poller duplicates the same expensive queries every 10s
The background SSE poller (`StartSSEPoller`) runs `CurrentProbeReadings` and `CurrentOutletStates` every 10 seconds — the exact same window function scans used by the dashboard API endpoints. This creates contention on DuckDB when a user loads the dashboard at the same time the poller fires.

### 4. Probe config auto-init writes during read requests
`HandleProbeList` calls `InitProbeConfig` (INSERT OR IGNORE) for any probe not yet in SQLite. This turns a read endpoint into a read+write, which means:
- SQLite WAL lock contention if another request is writing
- Unnecessary work on every load even when no new probes exist

### 5. No response caching at the API layer
Every request hits the database regardless of how recently the same data was served. The frontend has `staleTime` (10s for most queries), but the backend has no equivalent — it re-queries on every request within that stale window.

### 6. System status makes two separate DuckDB calls
`HandleSystemStatus` runs two independent DuckDB queries (`ControllerMeta` + `LastPollTime`) that could be combined into one.

---

## Improvement Plan

### Phase A: Quick Wins (no architectural changes)

#### A1. Cache feed status in-memory with TTL
**Effort:** 30 min | **Impact:** High (eliminates slowest endpoint)

Feed mode doesn't change frequently — it's set by the user and stays until changed or the next feeding cycle. Cache the result of `s.apex.Status()` for 15-30 seconds in an atomic value (`atomic.Value` or a mutex-protected struct with expiry).

```go
// In Server struct:
feedStatusCache struct {
    sync.RWMutex
    data   apex.StatusResponse
    expire time.Time
}

func (s *Server) HandleFeedGet(w, r) {
    if cached := s.getFeedStatusCached(); cached != nil && !cached.expired() {
        writeJSON(w, 200, cached.data)
        return
    }
    // ... fetch from Apex, cache result for 30s
}
```

Also invalidate the cache in `HandleFeedSet` after a successful mode change.

#### A2. Combine system status DuckDB queries
**Effort:** 15 min | **Impact:** Low-Medium

Merge `ControllerMeta` and `LastPollTime` into a single handler that makes fewer DB connections:

```go
// New combined query or just run both in sequence with one connection open
func (s *Server) HandleSystemStatus(w, r) {
    meta, _ := s.duck.ControllerMeta(ctx)     // already fast (LIMIT 1)
    lastPoll, _ := s.duck.LastPollTime(ctx)   // already fast (MAX aggregate)
    // These are both fast — the real win is reducing connection overhead
}
```

Actually these are already fast queries. The bigger win here is ensuring DuckDB's connection pool is warm. Consider opening DuckDB with `ReadConcurrency` set appropriately.

#### A3. Defer probe config auto-init to a background goroutine
**Effort:** 45 min | **Impact:** Medium

Move the `InitProbeConfig` loop out of the hot path:

```go
func (s *Server) HandleProbeList(w, r) {
    readings := s.duck.CurrentProbeReadings(ctx)
    configs := s.sqlite.ListProbeConfigs(ctx)
    
    // Fire-and-forget init for new probes — don't block the response
    go func() {
        cfgMap := buildConfigMap(configs)
        for _, rd := range readings {
            if _, exists := cfgMap[rd.Name]; !exists {
                cls := apex.ClassifyInput(...)
                s.sqlite.InitProbeConfig(context.Background(), ...)
            }
        }
    }()
    
    // Return response immediately with whatever config exists
}
```

Trade-off: new probes show default classification for one cycle until the background init completes. This is acceptable since `splitCamelCase` provides a reasonable display name fallback.

### Phase B: In-Memory State Cache (medium effort, high impact)

#### B1. Add an in-memory snapshot store updated by SSE poller
**Effort:** 2-3 hours | **Impact:** High

The SSE poller already runs `CurrentProbeReadings` and `CurrentOutletStates` every 10 seconds. Instead of just comparing JSON bytes for diffing, store the results in an atomic snapshot that API handlers can read:

```go
type Snapshot struct {
    sync.RWMutex
    Probes     []db.ProbeReading
    Outlets    []db.OutletState
    UpdatedAt  time.Time
}

// SSE poller writes:
func (s *Server) publishUpdates(ctx context.Context) {
    probes := s.duck.CurrentProbeReadings(ctx)
    outlets := s.duck.CurrentOutletStates(ctx)
    
    s.snapshot.Lock()
    s.snapshot.Probes = probes
    s.snapshot.Outlets = outlets
    s.snapshot.UpdatedAt = time.Now()
    s.snapshot.Unlock()
    
    // ... existing SSE broadcast logic
}

// API handlers read:
func (s *Server) HandleProbeList(w, r) {
    s.snapshot.RLock()
    probes := s.snapshot.Probes  // copy or reference
    updated := s.snapshot.UpdatedAt
    s.snapshot.RUnlock()
    
    configs := s.sqlite.ListProbeConfigs(ctx)
    // ... build response from in-memory data + SQLite config
}
```

**Benefits:**
- Eliminates DuckDB window function scans for dashboard loads entirely
- Data is at most 10 seconds stale (matches SSE poller interval) — acceptable for a dashboard
- Reduces DuckDB load, freeing it for the poller's write path
- No architectural changes to the frontend or API contract

**Risks:**
- If SSE poller crashes/stops, data becomes stale. Mitigation: add a staleness check in handlers — if snapshot is > 15s old, fall back to live query.
- Memory usage: probe readings are small (~20 probes × ~100 bytes = ~2KB). Negligible.

#### B2. Extend snapshot to include outlet states and system status
**Effort:** 30 min (on top of B1) | **Impact:** Medium

Same pattern — store `CurrentOutletStates` and the result of `ControllerMeta` + `LastPollTime` in the snapshot. The SSE poller already fetches outlets; adding controller meta is a cheap query to batch in.

### Phase C: Endpoint Consolidation (frontend-focused)

#### C1. Add a `/api/dashboard/summary` composite endpoint
**Effort:** 1-2 hours | **Impact:** Medium

Instead of 9 parallel requests, offer a single endpoint that returns everything the dashboard needs:

```typescript
// Frontend replaces 9 useQuery hooks with 1:
const { data } = useQuery({
    queryKey: qk.dashboard.summary,
    queryFn: getDashboardSummary,
    staleTime: 10_000,
})

// Backend combines all queries efficiently:
GET /api/dashboard/summary → {
    layout: DashboardItem[],
    probes: ProbeResponse[],      // from snapshot or live DuckDB
    outlets: OutletResponse[],    // from snapshot or live DuckDB  
    devices: Device[],            // SQLite (already fast)
    system: SystemStatus,         // from snapshot or live DuckDB
    feed: FeedStatus,             // from cache or live Apex
    livestockSummary: { sickCount: N, quarantineCount: M },  // aggregated, not full list!
    dueItems: DueItem[],          // SQLite (already fast)
}
```

**Key optimization:** The dashboard only needs `sickCount` and `quarantineCount` from livestock — not the full list. Change to an aggregate query:
```sql
SELECT COALESCE(SUM(CASE WHEN status = 'sick' THEN quantity ELSE 0 END), 0) as sick_count,
       COALESCE(SUM(CASE WHEN status = 'quarantine' THEN quantity ELSE 0 END), 0) as quarantine_count
FROM livestock
```

**Benefits:**
- Reduces HTTP overhead from 9 requests to 1
- Backend can batch SQLite queries in a single transaction/connection
- Eliminates the "waterfall" effect where React waits for all queries individually

**Trade-off:** Loses granular TanStack Query caching per resource. Mitigation: keep individual endpoints for other pages, use summary only on Dashboard page. Or split into 2 endpoints: one for live data (probes/outlets/feed) and one for static data (layout/devices/livestock/due).

### Phase D: DuckDB Query Optimization

#### D1. Add indexes to probe_readings and outlet_states
**Effort:** 30 min | **Impact:** Medium-High (if tables are large)

The window function `ROW_NUMBER() OVER (PARTITION BY probe_did ORDER BY ts DESC)` benefits from an index on `(probe_did, ts DESC)`. DuckDB supports indexes but they're created differently than SQLite:

```sql
CREATE INDEX idx_probe_readings_did_ts ON probe_readings(probe_did, ts DESC);
CREATE INDEX idx_outlet_states_did_ts ON outlet_states(outlet_did, ts DESC);
```

Check if these already exist in the schema. If not, add them during bootstrap/migration.

#### D2. Consider materialized views for current readings
**Effort:** 1-2 hours | **Impact:** High (if tables grow large)

DuckDB supports materialized views that can be refreshed incrementally:

```sql
CREATE OR REPLACE VIEW current_probe_readings AS
SELECT ts, probe_did, probe_type, probe_name, value
FROM (
    SELECT *, ROW_NUMBER() OVER (PARTITION BY probe_did ORDER BY ts DESC) AS rn
    FROM probe_readings
) WHERE rn = 1;
```

The poller could refresh this view after each write cycle instead of the API server computing it on every read. However, DuckDB materialized views have limitations — test performance before committing.

---

## Recommended Implementation Order

| Priority | Item | Effort | Impact | Why first? |
|----------|------|--------|--------|------------|
| 1 | **A1** — Cache feed status | 30 min | High | Eliminates the slowest endpoint (external HTTP). Zero risk. |
| 2 | **B1** — In-memory snapshot from SSE poller | 2-3 hrs | High | Eliminates DuckDB window scans for dashboard loads. Reuses existing poller infrastructure. |
| 3 | **A3** — Defer probe config init | 45 min | Medium | Removes writes from read path. Simple change. |
| 4 | **C1** — Composite summary endpoint | 1-2 hrs | Medium | Reduces HTTP overhead. Only worthwhile after B1 makes individual queries fast. |
| 5 | **D1** — DuckDB indexes | 30 min | Depends on data size | Check table sizes first. If < 100K rows, impact is minimal. |

## Estimated Total Time: 4-6 hours for priorities 1-3 (quick wins + snapshot)

These three changes alone should reduce dashboard load time by 50-70% in typical scenarios, primarily by eliminating the Apex HTTP call and DuckDB window function scans.

---

## Metrics to Track

Before implementing, measure baseline:
1. Add `slog` timing to each handler: `s.log.Info("request", "handler", "probes", "duration_ms", elapsed.Milliseconds())`
2. Note which endpoint is the bottleneck in your specific setup (Apex latency varies by network)
3. Check DuckDB table sizes: `SELECT COUNT(*) FROM probe_readings; SELECT COUNT(*) FROM outlet_states;`

After implementing each phase, compare load times to validate improvement.
