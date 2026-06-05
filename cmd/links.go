package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"
)

var linksJSON bool

var linksCmd = &cobra.Command{
	Use:   "links [slug]",
	Short: "Show link graph stats (global or per-card)",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
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
	Inbound  int `json:"inbound"`
	Outbound int `json:"outbound"`
}

type cardLinksOutput struct {
	Slug     string   `json:"slug"`
	Inbound  []string `json:"inbound"`
	Outbound []string `json:"outbound"`
}

func runGlobalLinks() {
	graph, err := GetStore().LinkGraph()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if linksJSON {
		out := linkGraphOutput{Cards: make(map[string]cardLinkInfo)}
		for slug, stats := range graph {
			out.Cards[slug] = cardLinkInfo{Inbound: stats.Inbound, Outbound: stats.Outbound}
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
		if stats.Inbound == 0 {
			marker = " (ORPHAN)"
			orphans++
		} else if stats.Inbound >= 10 {
			marker = " (HUB)"
			hubs++
		}
		fmt.Printf("  %s: %d inbound, %d outbound%s\n", slug, stats.Inbound, stats.Outbound, marker)
	}

	fmt.Printf("\nSummary: %d cards, %d orphans, %d hubs\n", len(graph), orphans, hubs)
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
}
