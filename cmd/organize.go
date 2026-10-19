package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rickdgeerling/zettel/internal/store"
	"github.com/spf13/cobra"
)

var (
	organizeSince string
	organizeJSON  bool
)

var organizeCmd = &cobra.Command{
	Use:   "organize",
	Short: "Run maintenance checks and emit an organize report",
	Long: `Emits an organize report: orphans, hubs, unresolved conflicts,
recently modified cards (with their linked neighbors), and broken links.

Use --since YYYY-MM-DD to limit the recently-modified section to cards changed
since the last organize run.`,
	Run: func(cmd *cobra.Command, args []string) {
		since := time.Time{}
		if organizeSince != "" {
			t, err := time.Parse("2006-01-02", organizeSince)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: invalid --since date %q (use YYYY-MM-DD)\n", organizeSince)
				os.Exit(1)
			}
			since = t
		}

		report, err := GetStore().OrganizeReport(since)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if organizeJSON {
			data, err := json.MarshalIndent(report, "", "  ")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: failed to marshal report: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(string(data))
			stampLastOrganize()
			return
		}

		printOrganizeReport(report, since)
		stampLastOrganize()
	},
}

// stampLastOrganize records the run date so the next incremental run can
// use it as --since. Report output is already complete by the time this
// runs, so a stamping failure never eats the report.
func stampLastOrganize() {
	if err := GetStore().StampLastOrganize(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printOrganizeReport(report *store.OrganizeReport, since time.Time) {
	fmt.Println("Organize Report")
	fmt.Println("===============")
	fmt.Printf("Total active cards: %d\n\n", report.Total)

	fmt.Printf("Orphans (%d):\n", len(report.Orphans))
	if len(report.Orphans) == 0 {
		fmt.Println("  (none)")
	} else {
		for _, slug := range report.Orphans {
			fmt.Printf("  %s\n", slug)
		}
	}

	fmt.Printf("\nHubs (%d):\n", len(report.Hubs))
	if len(report.Hubs) == 0 {
		fmt.Println("  (none)")
	} else {
		for _, slug := range report.Hubs {
			fmt.Printf("  %s\n", slug)
		}
	}

	fmt.Printf("\nUnresolved Conflicts (%d):\n", len(report.Conflicts))
	if len(report.Conflicts) == 0 {
		fmt.Println("  (none)")
	} else {
		for _, slug := range report.Conflicts {
			fmt.Printf("  %s\n", slug)
		}
	}

	recentHeader := "\nRecently Modified"
	if !since.IsZero() {
		recentHeader += fmt.Sprintf(" (since %s)", since.Format("2006-01-02"))
	}
	fmt.Printf("%s (%d):\n", recentHeader, len(report.RecentlyModified))
	if len(report.RecentlyModified) == 0 {
		fmt.Println("  (none)")
	} else {
		for _, rc := range report.RecentlyModified {
			fmt.Printf("  %s (modified %s) — out: %s | in: %s\n",
				rc.Slug, rc.Modified, markNeighbors(rc.Outbound, "→"), markNeighbors(rc.Inbound, "←"))
		}
	}

	fmt.Printf("\nBroken Links (%d):\n", len(report.BrokenLinks))
	if len(report.BrokenLinks) == 0 {
		fmt.Println("  (none)")
	} else {
		for _, b := range report.BrokenLinks {
			fmt.Printf("  %s\n", formatBrokenLink(b))
		}
	}
}

// markNeighbors renders a neighbor list with direction markers (→ out,
// ← in).
func markNeighbors(slugs []string, marker string) string {
	if len(slugs) == 0 {
		return "(none)"
	}
	parts := make([]string, len(slugs))
	for i, s := range slugs {
		parts[i] = marker + s
	}
	return strings.Join(parts, ", ")
}

// formatBrokenLink renders one broken-link line, including the "did you
// mean" suggestion when one exists.
func formatBrokenLink(b store.BrokenLink) string {
	if b.Suggestion != "" {
		return fmt.Sprintf("%s -> [[%s]] (%s, did you mean [[%s]]?)", b.From, b.To, b.Kind, b.Suggestion)
	}
	return fmt.Sprintf("%s -> [[%s]] (%s)", b.From, b.To, b.Kind)
}

func init() {
	organizeCmd.Flags().StringVar(&organizeSince, "since", "", "Only include recently modified cards since this date (YYYY-MM-DD)")
	organizeCmd.Flags().BoolVar(&organizeJSON, "json", false, "Output as JSON")
}
