package cli

import (
	"fmt"
	"strings"
	"text/tabwriter"
	"time"
)

func (h searchHuddl) valid() bool {
	_, err := time.Parse(time.RFC3339, h.Attributes.StartsAt)
	return h.Type == "huddl" && strings.TrimSpace(h.ID) != "" && strings.TrimSpace(h.Attributes.Title) != "" && err == nil
}

func writeHuddlTable(output *strings.Builder, huddlz []searchHuddl) {
	table := tabwriter.NewWriter(output, 0, 4, 2, ' ', 0)
	fmt.Fprintln(table, "ID\tTITLE\tSTARTS AT\tLOCATION")
	for _, h := range huddlz {
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\n", tableCell(h.ID), tableCell(h.Attributes.Title), tableCell(h.Attributes.StartsAt), tableCell(h.Attributes.PhysicalLocation))
	}
	table.Flush()
}
