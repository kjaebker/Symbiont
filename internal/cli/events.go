package cli

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

// NewEventsCmd returns the "events" command for browsing the audit event log.
func NewEventsCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "events",
		Short: "Browse the system audit event log",
		RunE: func(cmd *cobra.Command, args []string) error {
			kind, _ := cmd.Flags().GetString("kind")
			correlationID, _ := cmd.Flags().GetString("correlation-id")
			initiatedBy, _ := cmd.Flags().GetString("initiated-by")
			since, _ := cmd.Flags().GetString("since")
			limit, _ := cmd.Flags().GetInt("limit")

			path := fmt.Sprintf("/api/events?limit=%d&", limit)
			if kind != "" {
				path += "kind=" + kind + "&"
			}
			if correlationID != "" {
				path += "correlation_id=" + correlationID + "&"
			}
			if initiatedBy != "" {
				path += "initiated_by=" + initiatedBy + "&"
			}
			if since != "" {
				t, err := parseSince(since)
				if err != nil {
					return fmt.Errorf("invalid --since value %q: use a duration like 1h, 24h, 7d or an RFC3339 timestamp", since)
				}
				path += "since=" + t.Format(time.RFC3339) + "&"
			}

			var resp struct {
				Events []struct {
					ID            int64   `json:"id"`
					TS            string  `json:"ts"`
					Kind          string  `json:"kind"`
					PayloadJSON   string  `json:"payload_json"`
					CorrelationID *string `json:"correlation_id"`
				} `json:"events"`
			}
			if err := client.Get(cmd.Context(), path, &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			if len(resp.Events) == 0 {
				fmt.Println("No events found.")
				return nil
			}

			headers := []string{"ID", "KIND", "SUMMARY", "TIME"}
			rows := make([][]string, 0, len(resp.Events))
			for _, e := range resp.Events {
				rows = append(rows, []string{
					fmt.Sprintf("%d", e.ID),
					e.Kind,
					auditSummary(e.Kind, e.PayloadJSON),
					formatTimestamp(e.TS),
				})
			}
			PrintTable(headers, rows)
			return nil
		},
	}

	cmd.Flags().String("kind", "", "filter by event kind (e.g. outlet_changed, alert_fired)")
	cmd.Flags().String("correlation-id", "", "filter by entity ID (outlet ID, livestock ID, etc.)")
	cmd.Flags().String("initiated-by", "", "filter by initiator (ui, cli, mcp, api)")
	cmd.Flags().String("since", "", "only show events after this time (duration like 1h, 24h, 7d or RFC3339)")
	cmd.Flags().Int("limit", 50, "max events to return (max 500)")
	return cmd
}

// parseSince accepts a duration string (1h, 24h, 7d) or RFC3339 timestamp.
func parseSince(s string) (time.Time, error) {
	// Try duration first.
	if len(s) > 0 {
		unit := s[len(s)-1]
		switch unit {
		case 'h', 'm', 's':
			d, err := time.ParseDuration(s)
			if err == nil {
				return time.Now().Add(-d), nil
			}
		case 'd':
			var days int
			if _, err := fmt.Sscanf(s[:len(s)-1], "%d", &days); err == nil {
				return time.Now().AddDate(0, 0, -days), nil
			}
		}
	}
	// Fall back to RFC3339.
	return time.Parse(time.RFC3339, s)
}

// auditSummary returns a short human-readable description of an audit event.
func auditSummary(kind, payloadJSON string) string {
	var w struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal([]byte(payloadJSON), &w); err != nil || w.Data == nil {
		return ""
	}
	d := w.Data
	str := func(k string) string {
		if v, ok := d[k].(string); ok {
			return v
		}
		return ""
	}
	switch kind {
	case "outlet_changed":
		return fmt.Sprintf("%s: %s → %s", str("name"), str("prev_state"), str("new_state"))
	case "feed_mode_activated":
		return "Feed " + str("mode")
	case "alert_fired":
		return fmt.Sprintf("%s — %s", str("rule_name"), str("probe_name"))
	case "alert_cleared":
		return str("rule_name") + " cleared"
	case "observation_recorded":
		name := str("livestock_name")
		if s := str("status"); s != "" {
			return name + " → " + s
		}
		return name
	case "livestock_added":
		return fmt.Sprintf("%s (%s)", str("name"), str("type"))
	case "livestock_updated":
		return str("name")
	case "journal_entry_created":
		return fmt.Sprintf("%s [%s]", str("category"), str("source"))
	case "auth_event":
		return fmt.Sprintf("%s: %s", str("auth_kind"), str("actor"))
	case "config_changed":
		return str("key")
	default:
		return ""
	}
}
