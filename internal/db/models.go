package db

import "time"

// ProbeReading represents a single probe reading row from DuckDB.
type ProbeReading struct {
	Timestamp time.Time
	DID       string
	Type      string
	Name      string
	Value     float64
}

// OutletState represents a single outlet state row from DuckDB.
type OutletState struct {
	Timestamp time.Time
	DID       string
	ID        int
	Name      string
	Type      string
	State     string
	Intensity int
}

// DataPoint represents a single time-series data point for charting.
type DataPoint struct {
	Timestamp time.Time
	Value     float64
}

// ControllerMeta represents the most recent controller metadata snapshot.
type ControllerMeta struct {
	Timestamp time.Time
	Serial    string
	Hostname  string
	Software  string
	Hardware  string
	Type      string
	Timezone  string
}
