# Apex API Notes
> Captured via Chrome DevTools — AOS firmware version: ___________
> Capture date: ___________
> Apex serial: ___________

This document is ground truth for the Apex local API as it behaves on this specific unit and firmware version.
Community docs and third-party integrations may differ. Always defer to what's captured here.

**Status:** The sections below are pre-filled with community-sourced best guesses (from `itchannel/apex-ha`,
Reef2Reef threads, and the Telegraf Neptune plugin). Fields marked `[VERIFY]` must be confirmed
against your actual unit using Chrome DevTools before implementing the apex client.

---

## 1. Login Endpoint

### Request

```
Method:  POST
URL:     http://<apex-ip>/rest/login     [VERIFY — may include port]
```

**Request Headers:**
```
Content-Type: application/json
```

**Request Body (exact JSON):**
```json
{
  "login": "<username>",
  "password": "<password>",
  "remember_me": false
}
```
> [VERIFY] Field names `login` and `password` — some sources show `user`/`pass`. Capture exact keys from DevTools.

### Response

```
Status code: 200    [VERIFY]
```

**Response Headers:**
```
Set-Cookie:  connect.sid=<value>; Path=/; HttpOnly    [VERIFY cookie name and flags]
```

**Response Body:**
```json
[VERIFY — may be empty or contain a success envelope]
```

### Notes
- Cookie name to use in subsequent requests: `connect.sid`    [VERIFY]
- Session appears to expire after: unknown — test by waiting or clearing cookies and watching for 401
- Any edge cases or unexpected behavior: [VERIFY]

---

## 2. Status Endpoint

### Request

```
Method:  GET
URL:     http://<apex-ip>/rest/status    [VERIFY]
```

**Request Headers:**
```
Cookie:  connect.sid=<session-value>
```

### Response

```
Status code: 200
Content-Type: application/json    [VERIFY]
```

**Full Response Body (save actual JSON — truncate or annotate as needed):**

> Save the complete raw JSON to `testdata/status-response.json`.
> Document the structure here.

**Top-level structure (community best guess):**
```json
{
  "system": { ... },
  "inputs":  [ ... ],
  "outputs": [ ... ]
}
```
> [VERIFY] exact top-level key names (`system`/`config`/`status`, `inputs`/`probes`, `outputs`/`outlets`)

**System / Controller fields:**
```
Field name in JSON   →   Description / notes
system.serial        →   Controller serial number
system.hostname      →   Configured hostname
system.software      →   Firmware/AOS version string           [VERIFY field name]
system.hardware      →   Hardware model string                 [VERIFY field name]
system.timezone      →   Timezone string                       [VERIFY]
system.date          →   Current controller time               [VERIFY format: Unix epoch or string?]
system.power_failed  →   Timestamp of last power failure       [VERIFY: "none" when no event]
system.power_restored→   Timestamp of last power restore       [VERIFY: "none" when no event]
```

**Inputs (probes) — array structure:**
```
Field name in JSON   →   Description / notes
name                 →   Probe name (e.g. "Temp", "pH", "ORP", "Sal")
value                →   Current reading as float or string     [VERIFY type]
unit                 →   Unit string (e.g. "°F", "mV", "ppt")  [VERIFY]
type                 →   Probe type string                      [VERIFY canonical values]
```

**Sample probe entry (fill in from actual response):**
```json
[VERIFY — paste a real entry here from testdata/status-response.json]
```

**Probe names and types present on this unit:**

| JSON Name | Value (example) | Unit | Type field value | Notes |
|---|---|---|---|---|
| [FILL FROM DEVTOOLS] | | | | |

**Outputs (outlets) — array structure:**
```
Field name in JSON   →   Description / notes
did                  →   Device ID string (e.g. "base_Var_1")   [VERIFY — may be "id" or integer]
name                 →   Human label
state                →   Outlet state string (see values below)
xstatus              →   Extended status for wireless devices (WAV/Vortech)  [VERIFY presence]
type                 →   Output type: "outlet", "variable", "alert", "virtual"
amp                  →   Current draw in amps (float)            [VERIFY field name: "amp" vs "amps"]
watt                 →   Power in watts (float)                  [VERIFY field name: "watt" vs "watts"]
```

**Sample outlet entry (fill in from actual response):**
```json
[VERIFY — paste a real entry here from testdata/status-response.json]
```

**Outlet ID format:** [VERIFY]
(Community sources show DID format like `"base_Var_1"`, `"base_Out_1"` — confirm exact format on your unit)

**Outlet state values observed (community sourced — VERIFY all):**
- Auto-ON state value: `"AON"`
- Auto-OFF state value: `"AOF"`
- Forced ON: `"ON"`
- Forced OFF: `"OFF"`
- Table program: `"TBL"`
- Program: `"PF1"`, `"PF2"`, etc.
- xstatus field values: `"OK"`, `"---"` [VERIFY]

**All outlets on this unit:**

| JSON ID/DID | Name | Notes |
|---|---|---|
| [FILL FROM DEVTOOLS] | | |

**Power event fields (in system/controller object):**
```
Field name for power failure:     system.power_failed    [VERIFY]
Field name for power restore:     system.power_restored  [VERIFY]
Timestamp format:                 [VERIFY: Unix epoch integer, string, or formatted date?]
Value when no event has occurred: "none"                 [VERIFY]
```

**Wireless device fields (WAV / Vortech xstatus), if present:**
```json
[VERIFY — paste from DevTools if WAV/Vortech devices are connected]
```

**Trident fields (Ca / Alk / Mg), if present:**
```json
[VERIFY — paste from DevTools if Trident is connected]
```

---

## 3. Outlets Endpoint (if separate from status)

### Request

```
Method:  GET
URL:     http://<apex-ip>/rest/outlets    [VERIFY if this exists separately from /rest/status]
```

**Does this endpoint exist separately from /rest/status?** [VERIFY]

**If yes — Response Body:**
```json
[VERIFY]
```

---

## 4. Outlet Control Endpoint

### Request

```
Method:  PUT
URL:     http://<apex-ip>/rest/outlets/<did>    [VERIFY — confirm DID format in URL]
```

**Request Headers:**
```
Content-Type: application/json
Cookie:       connect.sid=<session-value>
```

**Request Body — set to ON:**
```json
{"state": "ON"}    [VERIFY field name and value string]
```

**Request Body — set to OFF:**
```json
{"state": "OFF"}   [VERIFY]
```

**Request Body — set to AUTO:**
```json
{"state": "AUTO"}  [VERIFY — may be "AON"/"AOF" or just "AUTO"]
```

### Response

```
Status code on success:       200    [VERIFY — may be 204]
Status code on invalid state: 400    [VERIFY]
Status code on unknown outlet ID: 404 [VERIFY]
```

**Response Body on success:**
```json
[VERIFY — may echo the updated outlet object or be empty]
```

**Response Body on error:**
```json
[VERIFY]
```

---

## 5. Session Expiry / 401 Behavior

**How was 401 tested?** [VERIFY — manually clear cookies or wait for natural expiry]

**401 Response:**
```
Status code: 401
```

**401 Response Body:**
```json
[VERIFY]
```

**401 Response Headers:**
```
WWW-Authenticate: [VERIFY]
```

**Observed session lifetime:** [VERIFY]
**Re-auth flow:** Same POST to /rest/login? [VERIFY]

---

## 6. Legacy Endpoints (AOS 5+ compatibility)

**Does `GET /cgi-bin/status.xml` work?** [VERIFY]
**Does `GET /cgi-bin/status.json` work?** [VERIFY]

If yes, any fields present in legacy that are absent from `/rest/status`?
```
[VERIFY]
```

---

## 7. Other Endpoints Observed

Document any other endpoints seen in DevTools that may be useful:

| Method | URL | Description |
|---|---|---|
| GET | /rest/olog | Outlet event log (requires session) |
| [FILL FROM DEVTOOLS] | | |

---

## 8. Cross-Reference Notes

**Community Go client (`ApexRest` or similar) — differences from what's captured here:**
```
[FILL IN after DevTools capture]
```

**Home Assistant `itchannel/apex-ha` — differences:**
```
Community Python HA integration uses similar session cookie flow.
Verify field names match — HA code may use older AOS field names.
```

**Any firmware-version-specific behavior to be aware of:**
```
AOS 5+ uses /rest/* endpoints and connect.sid cookie auth.
Pre-AOS-5 units use basic auth and /cgi-bin/ endpoints — NOT supported by Symbiont.
```

---

## 9. Implementation Decisions

Based on this capture, note any decisions made for `internal/apex/`:

- Login body struct fields: `login`, `password`, `remember_me` (bool)    [VERIFY]
- Cookie extraction method: scan `Set-Cookie` response header for `connect.sid`
- Outlet ID type in Go (string vs int): string (DID format)    [VERIFY]
- State field name in PUT body: `state`    [VERIFY]
- Any probes to skip or handle specially: [FILL IN]
- Any outlets to skip or handle specially: skip `type == "virtual"` and `type == "alert"` if not needed    [VERIFY]
