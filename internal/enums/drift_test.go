package enums_test

import (
	"database/sql"
	"regexp"
	"sort"
	"testing"

	"github.com/kjaebker/symbiont/internal/db"
	"github.com/kjaebker/symbiont/internal/enums"
)

// parsedCheck holds values extracted from a single CHECK(col IN (...)) expression.
type parsedCheck struct {
	table  string
	column string
	values []string // sorted
}

// parseCheckConstraints queries sqlite_master and returns all IN-list CHECK
// constraints found across every table's CREATE TABLE statement.
func parseCheckConstraints(sqliteDB *sql.DB) ([]parsedCheck, error) {
	rows, err := sqliteDB.Query(`SELECT name, sql FROM sqlite_master WHERE type='table' AND sql IS NOT NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Matches CHECK(column IN ('v1','v2',...)) — values may span multiple lines.
	reCheck := regexp.MustCompile(`CHECK\((\w+)\s+IN\s+\(([^)]+)\)\)`)
	reVal := regexp.MustCompile(`'([^']+)'`)

	var result []parsedCheck
	for rows.Next() {
		var tableName, tableSql string
		if err := rows.Scan(&tableName, &tableSql); err != nil {
			return nil, err
		}
		for _, m := range reCheck.FindAllStringSubmatch(tableSql, -1) {
			var vals []string
			for _, vm := range reVal.FindAllStringSubmatch(m[2], -1) {
				vals = append(vals, vm[1])
			}
			sort.Strings(vals)
			result = append(result, parsedCheck{table: tableName, column: m[1], values: vals})
		}
	}
	return result, rows.Err()
}

// TestSchemaDrift asserts that each enum Set whose values mirror a SQLite
// CHECK(col IN (...)) constraint stays in sync with the schema. If a value is
// added to the schema constraint without updating the enum (or vice-versa),
// this test fails.
//
// Enums without a 1:1 schema CHECK are intentionally excluded:
//   - InitiatedBy — lives in event JSON payloads, not a column constraint
//   - HistoryIntervals — API-layer only, no DB equivalent
//   - ImageExtensions — API-layer only, no DB equivalent
//   - DeviceTypes — devices.device_type is untyped (user values tolerated)
//   - DosingProductLines — dosing_products.type has its own CHECK not mapped here
//   - TokenScopes — auth_tokens.scope has no CHECK constraint in the schema
//   - TankTypes — tank_profile.tank_type has no CHECK constraint in the schema
func TestSchemaDrift(t *testing.T) {
	sqliteDB, err := db.OpenSQLite(":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { sqliteDB.Close() })

	constraints, err := parseCheckConstraints(sqliteDB.DB())
	if err != nil {
		t.Fatalf("parse constraints: %v", err)
	}

	// Build a lookup keyed by "table.column".
	lookup := make(map[string][]string, len(constraints))
	for _, c := range constraints {
		lookup[c.table+"."+c.column] = c.values
	}

	type enumCheck struct {
		schemaKey string // "table.column"
		set       enums.Set
		enumName  string // for readable error messages
	}

	checks := []enumCheck{
		{"livestock.type", enums.LivestockTypes, "LivestockTypes"},
		{"livestock.status", enums.LivestockStatuses, "LivestockStatuses"},
		{"livestock_observations.status", enums.LivestockStatuses, "LivestockStatuses (livestock_observations)"},
		{"journal_entries.category", enums.JournalCategories, "JournalCategories"},
		{"journal_entries.sentiment", enums.JournalSentiments, "JournalSentiments"},
		{"journal_entries.source", enums.JournalSources, "JournalSources"},
		{"agent_settings.tone", enums.AgentTones, "AgentTones"},
		{"dashboard_items.item_type", enums.DashboardItemTypes, "DashboardItemTypes"},
	}

	for _, c := range checks {
		t.Run(c.enumName, func(t *testing.T) {
			schemaVals, ok := lookup[c.schemaKey]
			if !ok {
				t.Errorf("no CHECK(... IN ...) found for %s — schema changed?", c.schemaKey)
				return
			}
			enumVals := c.set.Values() // already sorted by NewSet
			if !slicesEqual(schemaVals, enumVals) {
				t.Errorf("enums.%s is out of sync with schema %s:\n  schema: %v\n  enum:   %v",
					c.enumName, c.schemaKey, schemaVals, enumVals)
			}
		})
	}
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
