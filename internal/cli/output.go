package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// PrintJSON marshals v as indented JSON to stdout.
func PrintJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

// PrintTable prints aligned columns to stdout.
func PrintTable(headers []string, rows [][]string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, strings.Join(headers, "\t"))
	for _, row := range rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	w.Flush()
}

// KV is a key-value pair for display.
type KV struct {
	Key   string
	Value string
}

// PrintKeyValue prints key-value pairs with right-aligned keys.
func PrintKeyValue(pairs []KV) {
	maxKey := 0
	for _, p := range pairs {
		if len(p.Key) > maxKey {
			maxKey = len(p.Key)
		}
	}
	for _, p := range pairs {
		if p.Key == "" {
			fmt.Println()
			continue
		}
		fmt.Printf("%*s:  %s\n", maxKey, p.Key, p.Value)
	}
}

// PrintSection prints a section header followed by indented key-value pairs.
func PrintSection(title string, pairs []KV) {
	fmt.Println(title)
	for _, p := range pairs {
		fmt.Printf("  %-12s %s\n", p.Key+":", p.Value)
	}
}

// IsJSON returns true if the --json flag is set.
func IsJSON(cmd *cobra.Command) bool {
	v, _ := cmd.Flags().GetBool("json")
	return v
}

// ANSI color helpers for terminal output.
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
)

// ColorStatus returns a color-coded status string.
func ColorStatus(status string) string {
	switch status {
	case "normal", "OK":
		return colorGreen + status + colorReset
	case "warning":
		return colorYellow + status + colorReset
	case "critical":
		return colorRed + status + colorReset
	default:
		return status
	}
}

// ColorState returns a color-coded outlet state string.
func ColorState(state string) string {
	switch strings.ToUpper(state) {
	case "ON", "AON":
		return colorGreen + state + colorReset
	case "OFF", "AOF":
		return colorRed + state + colorReset
	case "AUTO":
		return colorBlue + state + colorReset
	default:
		return state
	}
}
