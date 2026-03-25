# Symbiont — Device Entity

## Context

Display names are currently configured separately for each probe and outlet. The Settings page already manually syncs outlet names to power probes (Settings.tsx:688-694), and `groupPowerPairs()` in Dashboard.tsx detects W/A pairs by naming convention. This implicit grouping should be made explicit via a Device entity that:

- Serves as single source of truth for display name (set once, cascades to outlet + probes)
- Holds rich metadata (description, brand, model, notes, images) for equipment tracking
- Links one outlet + zero or more probes
- Becomes the drag-and-drop unit in the future layout builder (Phase 7)

**Key design choice: write-through naming with unit suffixes.** When a device name changes, the API writes it to `outlet_config.display_name` (bare name) and `probe_config.display_name` (name + unit suffix). For example, a device named "Heater" produces:

- **Outlet:** "Heater"
- **Watts probe:** "Heater (watts)"
- **Amps probe:** "Heater (amps)"
- **Temp probe:** "Heater (°F)"

The suffix is derived from the probe's type via the existing `probeTypeToUnit()` function in `internal/api/probes.go`, lowercased. This means the 7+ existing display name resolution callsites (probes.go, outlets.go, alerts.go, export.go, engine.go, etc.) keep working with zero changes — they already read from probe_config/outlet_config.

---

## Schema

### New table: `devices`

```sql
CREATE TABLE IF NOT EXISTS devices (
    id            INTEGER  PRIMARY KEY AUTOINCREMENT,
    name          TEXT     NOT NULL,
    device_type   TEXT,
    description   TEXT,
    brand         TEXT,
    model         TEXT,
    notes         TEXT,
    image_path    TEXT,
    outlet_id     TEXT     UNIQUE,
    display_order INTEGER  NOT NULL DEFAULT 999,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

- `device_type` — free-text with a known set enforced at the app layer: `heater`, `pump`, `wavemaker`, `light`, `skimmer`, `reactor`, `doser`, `ato`, `chiller`, `fan`, `other`

- `outlet_id` is UNIQUE — one device per outlet (nullable for probe-only devices like a standalone temp sensor)
- `image_path` stores a relative path to `{data_dir}/images/`

### Migration: add `device_id` to `probe_config`

```sql
ALTER TABLE probe_config ADD COLUMN device_id INTEGER REFERENCES devices(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_probe_config_device ON probe_config(device_id);
```

Run via the existing `CreateSQLiteSchema` function. SQLite `ALTER TABLE ADD COLUMN` is safe; catch "duplicate column" error to make it idempotent.

### Go struct (`internal/db/sqlite_models.go`)

```go
type Device struct {
    ID           int64     `json:"id"`
    Name         string    `json:"name"`
    DeviceType   *string   `json:"device_type"`
    Description  *string   `json:"description"`
    Brand        *string   `json:"brand"`
    Model        *string   `json:"model"`
    Notes        *string   `json:"notes"`
    ImagePath    *string   `json:"image_path"`
    OutletID     *string   `json:"outlet_id"`
    DisplayOrder int       `json:"display_order"`
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}
```

Add `DeviceID *int64 json:"device_id"` to existing `ProbeConfig` struct.

---

## DB Query Functions (`internal/db/sqlite_queries.go`)

```go
func (s *SQLiteDB) ListDevices(ctx) ([]Device, error)
func (s *SQLiteDB) GetDevice(ctx, id int64) (*Device, error)
func (s *SQLiteDB) InsertDevice(ctx, d Device) (int64, error)
func (s *SQLiteDB) UpdateDevice(ctx, id int64, d Device) error
func (s *SQLiteDB) DeleteDevice(ctx, id int64) error
func (s *SQLiteDB) ListDeviceProbes(ctx, deviceID int64) ([]ProbeConfig, error)
func (s *SQLiteDB) SetProbeDevice(ctx, probeName string, deviceID *int64) error
func (s *SQLiteDB) SyncDeviceDisplayNames(ctx, deviceID int64, name string) error
```

`SyncDeviceDisplayNames` does the write-through in a transaction. The outlet gets the bare device name. Each probe gets the device name + a unit suffix derived from the probe's type (looked up from DuckDB via the probe list). The function signature becomes:

```go
func (s *SQLiteDB) SyncDeviceDisplayNames(ctx, deviceID int64, name string, probeUnits map[string]string) error
```

Where `probeUnits` maps probe name → lowercased unit string (e.g. `"ReturnPmpW" → "watts"`). The caller (API handler) builds this map using `probeTypeToUnit()` from the probe list data.

```sql
-- Outlet: bare name
UPDATE outlet_config SET display_name = ? WHERE outlet_id = (SELECT outlet_id FROM devices WHERE id = ?);
-- Each probe: name + unit suffix, e.g. "Heater (watts)"
UPDATE probe_config SET display_name = ? WHERE probe_name = ? AND device_id = ?;
```

Update existing `ProbeConfig` scan calls to include `device_id` column.

---

## API Endpoints (`internal/api/devices.go`, new file)

| Method | Path | Handler |
|--------|------|---------|
| `GET` | `/api/devices` | List all devices with `probe_names[]` |
| `GET` | `/api/devices/suggestions` | Auto-detect groupings from Apex naming |
| `POST` | `/api/devices` | Create device, link outlet + probes, sync names |
| `GET` | `/api/devices/{id}` | Get single device with probe membership |
| `PUT` | `/api/devices/{id}` | Update device; if name changed, sync names |
| `DELETE` | `/api/devices/{id}` | Delete device (ON DELETE SET NULL clears probe links) |
| `PUT` | `/api/devices/{id}/probes` | Replace probe membership list |
| `POST` | `/api/devices/{id}/image` | Upload image (multipart, JPEG/PNG/WebP, 5MB max) |
| `DELETE` | `/api/devices/{id}/image` | Remove image file and clear `image_path` |

Register in `server.go:registerRoutes`. Note: `/api/devices/suggestions` must be registered BEFORE `/api/devices/{id}` to avoid routing conflict.

### Suggestions endpoint

Activates the dead-code `CorrelateOutletPower()` from `internal/apex/parser.go`. Loads current probes and outlets from DuckDB, runs correlation, returns suggested groupings with `splitCamelCase(outletName)` as the suggested display name. Excludes outlets that already have a device.

### Response shape

```json
{
  "devices": [{
    "id": 1,
    "name": "Return Pump",
    "device_type": "pump",
    "description": "Main return pump",
    "brand": "Sicce",
    "model": "Syncra 5.0",
    "outlet_id": "base_Outlet3",
    "probe_names": ["ReturnPmpW", "ReturnPmpA"],
    ...
  }]
}
```

### Name conflict guard

`HandleProbeConfigUpdate` and `HandleOutletConfigUpdate` — if the entity is linked to a device, reject `display_name` changes with a 409 error: `"display name is managed by device '{name}'"`. This prevents drift.

### Image storage

Images saved to `{data_dir}/images/device-{id}-{timestamp}.{ext}`. Data dir defaults to the directory containing the SQLite file. Created on first upload. Served via a static file handler at `GET /data/images/*`.

---

## Frontend

### Types (`frontend/src/api/types.ts`)

```typescript
interface Device {
  id: number
  name: string
  device_type: string | null
  description: string | null
  brand: string | null
  model: string | null
  notes: string | null
  image_path: string | null
  outlet_id: string | null
  display_order: number
  created_at: string
  updated_at: string
  probe_names: string[]
}

interface DeviceSuggestion {
  outlet_name: string
  outlet_id: string
  probe_names: string[]
  suggested_name: string
}
```

### API client (`frontend/src/api/client.ts`)

Standard functions: `getDevices`, `createDevice`, `updateDevice`, `deleteDevice`, `setDeviceProbes`, `uploadDeviceImage`, `deleteDeviceImage`, `getDeviceSuggestions`.

### Hooks (`frontend/src/hooks/useDevices.ts`, new)

Standard TanStack Query hooks: `useDevices`, `useCreateDevice`, `useUpdateDevice`, `useDeleteDevice`, `useSetDeviceProbes`, `useDeviceSuggestions`.

### Settings — Devices tab (`frontend/src/pages/Settings.tsx`)

Add a "Devices" tab alongside existing Dashboard/Probes/Outlets tabs:

- **Device list** — Table or card grid showing all devices with name, type, brand/model, linked outlet, probe count
- **Add Device** — Form with: name, type (dropdown), description, brand, model, notes, outlet selector (dropdown of unlinked outlets), probe multi-select (checkboxes of available probes), image upload
- **Suggest Devices** button — Calls suggestions endpoint, presents as pre-filled cards the user can accept
- **Edit** — Inline or modal edit; changing name shows note: "Name applies to linked outlet and probes (probes get unit suffix, e.g. 'Heater (watts)')"
- **Delete** — Confirmation dialog; explains that probe/outlet configs are kept, just unlinked

### Dashboard impact

Minimal. The write-through naming means `groupPowerPairs()` in Dashboard.tsx already gets correct display names from the probe list endpoint. No changes needed unless we want to add device-level grouping later.

---

## Migration Path

Purely additive — no data migration needed:
1. Schema adds `devices` table and `device_id` column (nullable)
2. Existing probe_config/outlet_config rows work unchanged (`device_id = NULL` = standalone)
3. Users create devices via Settings or accept suggestions
4. Probes/outlets not linked to devices continue with their individual display names

---

## Layout Builder Integration (deferred to Phase 7)

The device becomes a natural node type in the layout builder:

```json
{ "type": "device", "data": { "device_id": 1 } }
```

Renders as a compound card: device image + outlet state toggle + probe values. Standalone probes/outlets remain as individual node types. This is not part of this PR — just noting the design alignment.

---

## Implementation Order

| Step | Scope | Files |
|------|-------|-------|
| 1. Schema + models | Backend | `sqlite_schema.go`, `sqlite_models.go` |
| 2. DB queries + tests | Backend | `sqlite_queries.go`, new test file |
| 3. API handlers + tests | Backend | `devices.go` (new), `server.go`, `config.go` (name guard), `devices_test.go` (new) |
| 4. Image upload + serving | Backend | `devices.go`, `server.go` |
| 5. Frontend types + client + hooks | Frontend | `types.ts`, `client.ts`, `useDevices.ts` (new) |
| 6. Settings Devices tab | Frontend | `Settings.tsx` |

Steps 1-2, 3-4, and 5-6 are natural commit boundaries.

---

## Verification

```bash
# Backend
go build ./...
go test ./...

# Frontend
cd frontend && npx tsc --noEmit && npm test

# Manual
# 1. GET /api/devices/suggestions → shows auto-detected groupings
# 2. POST /api/devices with name + outlet_id + probe_names → creates device
# 3. GET /api/probes → linked probes show "DeviceName (unit)" as display_name
# 4. GET /api/outlets → linked outlet shows bare device name as display_name
# 5. PUT /api/config/probes/{name} with display_name → rejected with 409 if linked to device
# 6. Upload image → file saved, path stored, retrievable
# 7. Delete device → probes/outlets unlinked but configs preserved
```

---

## Key Files

- `internal/db/sqlite_schema.go` — DDL for devices table + migration
- `internal/db/sqlite_models.go` — Device struct, ProbeConfig.DeviceID
- `internal/db/sqlite_queries.go` — Device CRUD, SyncDeviceDisplayNames, SetProbeDevice
- `internal/api/devices.go` (new) — All device HTTP handlers
- `internal/api/config.go` — Add name conflict guard
- `internal/api/server.go` — Register device routes
- `internal/apex/parser.go` — Activate CorrelateOutletPower for suggestions
- `frontend/src/api/types.ts` — Device, DeviceSuggestion types
- `frontend/src/api/client.ts` — Device API client functions
- `frontend/src/hooks/useDevices.ts` (new) — TanStack Query hooks
- `frontend/src/pages/Settings.tsx` — Devices tab
