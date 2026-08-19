package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/rickdgeerling/zettel/internal/store"
	"github.com/spf13/cobra"
)

var (
	linksJSON            bool
	linksIncludeArchived bool
	linksBroken          bool
)

var linksCmd = &cobra.Command{
	Use:   "links [slug]",
	Short: "Show link graph stats (global or per-card), or broken links with --broken",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if linksBroken {
			if len(args) > 0 {
				fmt.Fprintf(os.Stderr, "Error: --broken does not take a slug argument\n")
				os.Exit(1)
			}
			runBrokenLinks()
			return
		}
		if len(args) == 0 {
			runGlobalLinks()
		} else {
			runCardLinks(args[0])
		}
	},
}

type linkGraphOutput struct {
	Cards map[string]cardLinkInfo `json:"cards"`
}

type cardLinkInfo struct {
	Inbound  int  `json:"inbound"`
	Outbound int  `json:"outbound"`
	Archived bool `json:"archived,omitempty"`
}

type cardLinksOutput struct {
	Slug     string   `json:"slug"`
	Inbound  []string `json:"inbound"`
	Outbound []string `json:"outbound"`
}

type brokenLinksOutput struct {
	Broken []brokenLinkInfo `json:"broken"`
}

type brokenLinkInfo struct {
	From string `json:"from"`
	To   string `json:"to"`
	Kind string `json:"kind"`
}

func runGlobalLinks() {
	var graph map[string]store.LinkStats
	var err error
	if linksIncludeArchived {
		graph, err = GetStore().LinkGraphIncludingArchived()
	} else {
		graph, err = GetStore().LinkGraph()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if linksJSON {
		out := linkGraphOutput{Cards: make(map[string]cardLinkInfo)}
		for slug, stats := range graph {
			out.Cards[slug] = cardLinkInfo{Inbound: stats.Inbound, Outbound: stats.Outbound, Archived: stats.Archived}
		}
		data, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(data))
		return
	}

	fmt.Println("Link Graph:")
	fmt.Println("----------")

	slugs := make([]string, 0, len(graph))
	for slug := range graph {
		slugs = append(slugs, slug)
	}
	sort.Strings(slugs)

	var orphans, hubs int
	for _, slug := range slugs {
		stats := graph[slug]
		marker := ""
		if stats.Archived {
			marker = " (ARCHIVED)"
		} else if stats.Inbound == 0 {
			marker = " (ORPHAN)"
			orphans++
		} else if stats.Inbound >= 10 {
			marker = " (HUB)"
			hubs++
		}
		fmt.Printf("  %s: %d inbound, %d outbound%s\n", slug, stats.Inbound, stats.Outbound, marker)
	}

	if linksIncludeArchived {
		fmt.Printf("\nSummary: %d cards (including archived), %d orphans, %d hubs\n", len(graph), orphans, hubs)
	} else {
		fmt.Printf("\nSummary: %d active cards, %d orphans, %d hubs\n", len(graph), orphans, hubs)
	}
}

func runBrokenLinks() {
	broken, err := GetStore().BrokenLinks()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if linksJSON {
		out := brokenLinksOutput{Broken: make([]brokenLinkInfo, 0, len(broken))}
		for _, b := range broken {
			out.Broken = append(out.Broken, brokenLinkInfo{From: b.From, To: b.To, Kind: b.Kind})
		}
		data, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(data))
		return
	}

	if len(broken) == 0 {
		fmt.Println("No broken links")
		return
	}

	fmt.Println("Broken Links:")
	fmt.Println("-------------")
	for _, b := range broken {
		fmt.Printf("  %s -> [[%s]] (%s)\n", b.From, b.To, b.Kind)
	}
	fmt.Printf("\nSummary: %d broken links\n", len(broken))
}

func runCardLinks(slug string) {
	inbound, outbound, err := GetStore().LinksForCard(slug)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if linksJSON {
		out := cardLinksOutput{Slug: slug, Inbound: inbound, Outbound: outbound}
		data, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(data))
		return
	}

	fmt.Printf("Card: %s\n", slug)
	fmt.Printf("  Inbound (%d): %v\n", len(inbound), inbound)
	fmt.Printf("  Outbound (%d): %v\n", len(outbound), outbound)
}

func init() {
	linksCmd.Flags().BoolVar(&linksJSON, "json", false, "Output as JSON")
	linksCmd.Flags().BoolVar(&linksIncludeArchived, "include-archived", false, "Include archived cards in the graph")
	linksCmd.Flags().BoolVar(&linksBroken, "broken", false, "List [[wikilinks]] pointing to missing or archived cards")
}
