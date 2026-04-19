package db

// DosingProductSeeds returns a starter product catalog based on the configured
// dosing product line. Called once on startup when the table is empty.
func DosingProductSeeds(productLine string) []DosingProduct {
	switch productLine {
	case "brs_pharma":
		return []DosingProduct{
			{Brand: "BRS", Name: "2-Part Calcium", Type: "calcium", Unit: "mL"},
			{Brand: "BRS", Name: "2-Part Alkalinity", Type: "alkalinity", Unit: "mL"},
			{Brand: "BRS", Name: "Pharma Magnesium", Type: "magnesium", Unit: "mL"},
			{Brand: "BRS", Name: "Trace Elements A", Type: "trace", Unit: "mL"},
			{Brand: "BRS", Name: "Trace Elements B", Type: "trace", Unit: "mL"},
			{Brand: "BRS", Name: "Pharma Amino Acids", Type: "amino", Unit: "mL"},
		}
	case "red_sea":
		return []DosingProduct{
			{Brand: "Red Sea", Name: "Reef Energy A", Type: "amino", Unit: "mL"},
			{Brand: "Red Sea", Name: "Reef Energy B", Type: "amino", Unit: "mL"},
			{Brand: "Red Sea", Name: "Foundation A (Calcium)", Type: "calcium", Unit: "mL"},
			{Brand: "Red Sea", Name: "Foundation B (Alkalinity)", Type: "alkalinity", Unit: "mL"},
			{Brand: "Red Sea", Name: "Foundation C (Magnesium)", Type: "magnesium", Unit: "mL"},
			{Brand: "Red Sea", Name: "Trace Colors A", Type: "trace", Unit: "mL"},
			{Brand: "Red Sea", Name: "Trace Colors B", Type: "trace", Unit: "mL"},
			{Brand: "Red Sea", Name: "Trace Colors C", Type: "trace", Unit: "mL"},
			{Brand: "Red Sea", Name: "Trace Colors D", Type: "trace", Unit: "mL"},
		}
	case "tropic_marin":
		return []DosingProduct{
			{Brand: "Tropic Marin", Name: "Bio-Calcium", Type: "calcium", Unit: "mL"},
			{Brand: "Tropic Marin", Name: "Bio-Actif", Type: "amino", Unit: "mL"},
			{Brand: "Tropic Marin", Name: "K-Balance", Type: "trace", Unit: "mL"},
			{Brand: "Tropic Marin", Name: "Strontium", Type: "trace", Unit: "mL"},
		}
	default:
		// generic / none — provide common universal products
		return []DosingProduct{
			{Brand: "Generic", Name: "2-Part Part A (Calcium)", Type: "two_part_a", Unit: "mL"},
			{Brand: "Generic", Name: "2-Part Part B (Alkalinity)", Type: "two_part_b", Unit: "mL"},
			{Brand: "Generic", Name: "Magnesium", Type: "magnesium", Unit: "mL"},
			{Brand: "Generic", Name: "Trace Elements", Type: "trace", Unit: "mL"},
			{Brand: "Generic", Name: "Beneficial Bacteria", Type: "bacteria", Unit: "mL"},
		}
	}
}
