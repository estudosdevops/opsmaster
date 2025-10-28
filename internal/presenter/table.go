package presenter

import (
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
)

// PrintTable formats and prints data in a table to the console with rounded Unicode borders.
// It receives the header and rows as slices of strings.
//
// Uses the modern tablewriter API with StyleRounded for professional rounded borders:
//
//	╭─────────────┬─────────┬────────╮
//	│ INSTANCE ID │ ACCOUNT │ STATUS │
//	├─────────────┼─────────┼────────┤
//	│ i-123       │ 111111  │ ✅     │
//	╰─────────────┴─────────┴────────╯
//
// This implementation uses tablewriter.NewTable (modern API) instead of
// tablewriter.NewWriter (legacy API) to access advanced rendering features
// like rounded corners via tw.StyleRounded symbols.
func PrintTable(header []string, rows [][]string) {
	// Create table with rounded border style
	table := tablewriter.NewTable(os.Stdout,
		tablewriter.WithRenderer(renderer.NewBlueprint(tw.Rendition{
			Symbols: tw.NewSymbols(tw.StyleRounded),
		})),
		tablewriter.WithConfig(tablewriter.Config{
			Header: tw.CellConfig{
				Formatting: tw.CellFormatting{AutoFormat: tw.On},
				Alignment:  tw.CellAlignment{Global: tw.AlignLeft},
			},
			Row: tw.CellConfig{
				Alignment: tw.CellAlignment{Global: tw.AlignLeft},
			},
		}),
	)

	// Set header
	table.Header(header)

	// Add rows
	for _, row := range rows {
		_ = table.Append(row) // Error explicitly ignored (cosmetic operation)
	}

	// Render table
	_ = table.Render() // Error explicitly ignored (writes to stdout)
}
