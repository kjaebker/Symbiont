package db

// Probe status values, ordered by severity. They are derived from a probe's
// configured ranges and are the single source of truth for both the status
// colors in the UI and "probe_range" alert rules.
const (
	ProbeStatusUnknown  = "unknown"
	ProbeStatusNormal   = "normal"
	ProbeStatusWarning  = "warning"
	ProbeStatusCritical = "critical"
)

// AlertConditionProbeRange is the alert rule condition that follows the
// probe's configured ranges instead of carrying its own thresholds.
const AlertConditionProbeRange = "probe_range"

// ProbeStatus determines a probe's status from its config.
//
// Binary inputs with ok_value set are "normal" when the value matches it;
// otherwise alarms are "critical" and fluid sensors "warning". Analog probes
// are "critical" outside min/max_warning, "warning" outside min/max_normal.
// A probe with no config or no ranges is "unknown".
func ProbeStatus(value float64, cfg *ProbeConfig) string {
	if cfg == nil {
		return ProbeStatusUnknown
	}

	if cfg.IsBinary && cfg.OkValue != nil {
		if value == *cfg.OkValue {
			return ProbeStatusNormal
		}
		switch cfg.InputCategory {
		case "alarm":
			return ProbeStatusCritical
		case "fluid":
			return ProbeStatusWarning
		}
		return ProbeStatusUnknown
	}

	hasNormal := cfg.MinNormal != nil || cfg.MaxNormal != nil
	hasWarning := cfg.MinWarning != nil || cfg.MaxWarning != nil
	if !hasNormal && !hasWarning {
		return ProbeStatusUnknown
	}

	if cfg.MinWarning != nil && value < *cfg.MinWarning {
		return ProbeStatusCritical
	}
	if cfg.MaxWarning != nil && value > *cfg.MaxWarning {
		return ProbeStatusCritical
	}
	if cfg.MinNormal != nil && value < *cfg.MinNormal {
		return ProbeStatusWarning
	}
	if cfg.MaxNormal != nil && value > *cfg.MaxNormal {
		return ProbeStatusWarning
	}
	return ProbeStatusNormal
}

// ProbeStatusRank orders statuses so they can be compared with a severity:
// unknown and normal are 0, warning 1, critical 2.
func ProbeStatusRank(status string) int {
	switch status {
	case ProbeStatusWarning:
		return 1
	case ProbeStatusCritical:
		return 2
	default:
		return 0
	}
}

// CanReach reports whether this probe's config can ever produce the given
// severity ("warning" or "critical"). A probe_range alert rule on a probe
// that cannot reach its severity would never fire.
func (c *ProbeConfig) CanReach(severity string) bool {
	if c == nil {
		return false
	}
	if c.IsBinary && c.OkValue != nil {
		switch c.InputCategory {
		case "alarm":
			return true
		case "fluid":
			return severity == ProbeStatusWarning
		}
		return false
	}
	hasNormal := c.MinNormal != nil || c.MaxNormal != nil
	hasWarning := c.MinWarning != nil || c.MaxWarning != nil
	if severity == ProbeStatusCritical {
		return hasWarning
	}
	return hasNormal || hasWarning
}

// RangeFor returns the analog bounds whose breach produces the given
// severity: min/max_normal for "warning" (falling back to the warning bounds
// when no normal range is set), min/max_warning for "critical".
func (c *ProbeConfig) RangeFor(severity string) (low, high *float64) {
	if c == nil || c.IsBinary {
		return nil, nil
	}
	if severity == ProbeStatusCritical {
		return c.MinWarning, c.MaxWarning
	}
	if c.MinNormal != nil || c.MaxNormal != nil {
		return c.MinNormal, c.MaxNormal
	}
	return c.MinWarning, c.MaxWarning
}
