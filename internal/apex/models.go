package apex

import "encoding/json"

// StatusResponse is the top-level response from GET /rest/status.
type StatusResponse struct {
	System  SystemInfo    `json:"system"`
	Modules []Module      `json:"modules"`
	Nstat   NetworkStatus `json:"nstat"`
	Feed    FeedStatus    `json:"feed"`
	Power   PowerInfo     `json:"power"`
	Outputs []Output      `json:"outputs"`
	Inputs  []Input       `json:"inputs"`
	Link    LinkInfo      `json:"link"`
}

// SystemInfo contains controller identification and metadata.
type SystemInfo struct {
	Hostname string          `json:"hostname"`
	Software string          `json:"software"`
	Hardware string          `json:"hardware"`
	Serial   string          `json:"serial"`
	Type     string          `json:"type"`
	Extra    json.RawMessage `json:"extra"`
	Timezone string          `json:"timezone"`
	Date     int64           `json:"date"`
}

// PowerInfo contains timestamps for the last power failure and restore.
// These are Unix epoch integers at the top level of the status response,
// NOT nested under system.
type PowerInfo struct {
	Failed   int64 `json:"failed"`
	Restored int64 `json:"restored"`
}

// Input represents a probe reading from the Apex.
// There is no "unit" field — the Type field serves as the unit indicator.
//
// Observed types: "Temp", "pH", "Amps", "pwr", "volts", "digital"
//
// Per-outlet power draw is reported as Input entries, not as fields on Output.
// The naming convention is <OutletName>A for amps and <OutletName>W for watts.
type Input struct {
	DID   string  `json:"did"`
	Type  string  `json:"type"`
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

// Output represents an outlet/variable/alert/virtual device from the Apex.
//
// The Status field is a 4-element string array:
//
//	[0] state:     "AON", "AOF", "ON", "OFF", "TBL", "PF1"-"PF4"
//	[1] intensity: percentage as string (or empty for on/off outlets)
//	[2] health:    "OK" or "---"
//	[3] unknown:   always empty in observed responses
//
// Observed output types: "outlet", "variable", "alert", "virtual", "serial", "24v"
type Output struct {
	Status    []string `json:"status"`
	Intensity *int     `json:"intensity,omitempty"`
	Name      string   `json:"name"`
	GID       string   `json:"gid"`
	Type      string   `json:"type"`
	ID        int      `json:"ID"`
	DID       string   `json:"did"`
}

// State returns the outlet's current state string from status[0].
// Returns empty string if the status array is empty.
func (o Output) State() string {
	if len(o.Status) > 0 {
		return o.Status[0]
	}
	return ""
}

// Health returns the outlet's health string from status[2].
// Returns empty string if the status array has fewer than 3 elements.
func (o Output) Health() string {
	if len(o.Status) > 2 {
		return o.Status[2]
	}
	return ""
}

// Module represents a connected hardware module on the AquaBus.
type Module struct {
	ABAddr  int             `json:"abaddr"`
	HWType  string          `json:"hwtype"`
	HWRev   int             `json:"hwrev"`
	SWRev   int             `json:"swrev"`
	SWStat  string          `json:"swstat"`
	PCount  int64           `json:"pcount"`
	PGood   int64           `json:"pgood"`
	PError  int64           `json:"perror"`
	Reatt   int             `json:"reatt"`
	Inact   int             `json:"inact"`
	Boot    bool            `json:"boot"`
	Present bool            `json:"present"`
	Extra   json.RawMessage `json:"extra"`
}

// FeedStatus represents the current feed mode state.
// Name is the feed cycle number (0 = not feeding, 1-4 = feed cycle).
// Active indicates whether feed mode is currently engaged.
type FeedStatus struct {
	Name   int `json:"name"`
	Active int `json:"active"`
}

// NetworkStatus contains the controller's network configuration.
type NetworkStatus struct {
	DHCP           bool     `json:"dhcp"`
	Hostname       string   `json:"hostname"`
	IPAddr         string   `json:"ipaddr"`
	Netmask        string   `json:"netmask"`
	Gateway        string   `json:"gateway"`
	DNS            []string `json:"dns"`
	HTTPPort       int      `json:"httpPort"`
	FusionEnable   bool     `json:"fusionEnable"`
	Quality        int      `json:"quality"`
	Strength       int      `json:"strength"`
	Link           bool     `json:"link"`
	WifiAPLock     bool     `json:"wifiAPLock"`
	WifiEnable     bool     `json:"wifiEnable"`
	WifiAPPassword string   `json:"wifiAPPassword"`
	SSID           string   `json:"ssid"`
	WifiAP         bool     `json:"wifiAP"`
	UpdateFirmware bool     `json:"updateFirmware"`
	LatestFirmware string   `json:"latestFirmware"`
}

// LinkInfo contains the Apex Fusion cloud link status.
type LinkInfo struct {
	LinkState int    `json:"linkState"`
	LinkKey   string `json:"linkKey"`
	Link      bool   `json:"link"`
}

// FeedControlRequest is the body sent to PUT /rest/status/feed.
type FeedControlRequest struct {
	Name   int `json:"name"`   // 1-4 for Feed A-D; 0 to cancel
	Active int `json:"active"` // 1 to enable, 0 to cancel
}

// OutletState represents valid states for outlet control via
// PUT /rest/status/outputs/<did>.
type OutletState string

const (
	OutletOn  OutletState = "ON"
	OutletOff OutletState = "OFF"
)

// LoginRequest is the body sent to POST /rest/login.
type LoginRequest struct {
	Login      string `json:"login"`
	Password   string `json:"password"`
	RememberMe bool   `json:"remember_me"`
}

// LoginResponse is the body returned from POST /rest/login.
type LoginResponse struct {
	ConnectSID string `json:"connect.sid"`
}

// OutletControlRequest is the body sent to PUT /rest/status/outputs/<did>
// for safe runtime state toggling (preserves outlet programs).
type OutletControlRequest struct {
	DID    string   `json:"did"`
	Status []string `json:"status"`
	Type   string   `json:"type"`
}

// NewOutletControlRequest creates the request body for toggling an outlet.
func NewOutletControlRequest(did string, state OutletState) OutletControlRequest {
	return OutletControlRequest{
		DID:    did,
		Status: []string{string(state), "", "OK", ""},
		Type:   "outlet",
	}
}

