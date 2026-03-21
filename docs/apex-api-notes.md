# Apex API Notes
> Captured via Chrome DevTools — AOS firmware version: ___________
> Capture date: ___________
> Apex serial: ___________

This document is ground truth for the Apex local API as it behaves on this specific unit and firmware version. Community docs and third-party integrations may differ. Always defer to what's captured here.

---

## 1. Login Endpoint

### Request

```
Method:  POST
URL:     ___________________________________________
```

**Request Headers:**
```
Content-Type: ___________________________________________
Other:        ___________________________________________
```

**Request Body (exact JSON):**
```json

```

### Response

```
Status code: ___________
```

**Response Headers:**
```
Set-Cookie:  ___________________________________________
             (note cookie name, value format, Path, HttpOnly, expiry if present)
Other:       ___________________________________________
```

**Response Body:**
```json

```

### Notes
- Cookie name to use in subsequent requests: ___________
- Session appears to expire after: ___________ (if determinable)
- Any edge cases or unexpected behavior: ___________

---

## 2. Status Endpoint

### Request

```
Method:  GET
URL:     ___________________________________________
```

**Request Headers:**
```
Cookie:  ___________________________________________
Other:   ___________________________________________
```

### Response

```
Status code: ___________
Content-Type: ___________
```

**Full Response Body (save actual JSON — truncate or annotate as needed):**

> Save the complete raw JSON to `testdata/status-response.json`.
> Document the structure here.

**System / Controller fields:**
```
Field name in JSON   →   Description / notes
_______________________________________________________________
_______________________________________________________________
_______________________________________________________________
_______________________________________________________________
_______________________________________________________________
```

**Inputs (probes) — array structure:**
```
Field name in JSON   →   Description / notes
_______________________________________________________________
_______________________________________________________________
_______________________________________________________________
_______________________________________________________________
```

**Sample probe entry (fill in from actual response):**
```json

```

**Probe names and types present on this unit:**

| JSON Name | Value (example) | Unit | Type field value | Notes |
|---|---|---|---|---|
| | | | | |
| | | | | |
| | | | | |
| | | | | |
| | | | | |

**Outputs (outlets) — array structure:**
```
Field name in JSON   →   Description / notes
_______________________________________________________________
_______________________________________________________________
_______________________________________________________________
_______________________________________________________________
```

**Sample outlet entry (fill in from actual response):**
```json

```

**Outlet ID format:** ___________
(e.g. integer, string like "1_1", DID format like "base_Var_1" — note exactly)

**Outlet state values observed:**
- ON state value: ___________
- OFF state value: ___________
- AUTO state value: ___________
- xstatus field values: ___________

**All outlets on this unit:**

| JSON ID/DID | Name | Notes |
|---|---|---|
| | | |
| | | |
| | | |
| | | |
| | | |

**Power event fields (in system/controller object):**
```
Field name for power failure:  ___________
Field name for power restore:  ___________
Timestamp format:              ___________
Value when no event has occurred: ___________
```

**Wireless device fields (WAV / Vortech xstatus), if present:**
```json

```

**Trident fields (Ca / Alk / Mg), if present:**
```json

```

---

## 3. Outlets Endpoint (if separate from status)

### Request

```
Method:  GET
URL:     ___________________________________________
```

**Does this endpoint exist separately from /rest/status?** ___________

**If yes — Response Body:**
```json

```

---

## 4. Outlet Control Endpoint

### Request

```
Method:  PUT
URL:     ___________________________________________ (note ID format in URL)
```

**Request Headers:**
```
Content-Type: ___________________________________________
Cookie:       ___________________________________________
```

**Request Body — set to ON:**
```json

```

**Request Body — set to OFF:**
```json

```

**Request Body — set to AUTO:**
```json

```

### Response

```
Status code on success: ___________
Status code on invalid state: ___________
Status code on unknown outlet ID: ___________
```

**Response Body on success:**
```json

```

**Response Body on error:**
```json

```

---

## 5. Session Expiry / 401 Behavior

**How was 401 tested?** ___________________________________________

**401 Response:**
```
Status code: 401
```

**401 Response Body:**
```json

```

**401 Response Headers:**
```
WWW-Authenticate: ___________________________________________
Other:            ___________________________________________
```

**Observed session lifetime:** ___________
**Re-auth flow:** Same as initial login? ___________

---

## 6. Legacy Endpoints (AOS 5+ compatibility)

**Does `GET /cgi-bin/status.xml` work?** ___________
**Does `GET /cgi-bin/status.json` work?** ___________

If yes, any fields present in legacy that are absent from `/rest/status`?
```

```

---

## 7. Other Endpoints Observed

Document any other endpoints seen in DevTools that may be useful:

| Method | URL | Description |
|---|---|---|
| | | |
| | | |
| | | |

---

## 8. Cross-Reference Notes

**Community Go client (`ApexRest` or similar) — differences from what's captured here:**
```

```

**Home Assistant `itchannel/apex-ha` — differences:**
```

```

**Any firmware-version-specific behavior to be aware of:**
```

```

---

## 9. Implementation Decisions

Based on this capture, note any decisions made for `internal/apex/`:

- Login body struct fields: ___________
- Cookie extraction method: ___________
- Outlet ID type in Go (string vs int): ___________
- State field name in PUT body: ___________
- Any probes to skip or handle specially: ___________
- Any outlets to skip or handle specially: ___________
