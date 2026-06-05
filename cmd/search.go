package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	searchTag      []string
	searchCategory string
	searchStatus   string
	searchLimit    int
	searchOffset   int
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search cards by substring match and metadata filters",
	Run: func(cmd *cobra.Command, args []string) {
		query := strings.Join(args, " ")

		var catArg, statusArg *string
		if searchCategory != "" {
			catArg = &searchCategory
		}
		if searchStatus != "" {
			statusArg = &searchStatus
		}

		results, err := GetStore().SearchCards(query, searchTag, catArg, statusArg, int(searchLimit), int(searchOffset))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if len(results) == 0 {
			fmt.Println("No cards found")
			return
		}

		for _, r := range results {
			tags := ""
			if len(r.Tags) > 0 {
				tags = " [" + strings.Join(r.Tags, ", ") + "]"
			}
			fmt.Printf("%s | %s%s | cat=%s | status=%s | created=%s modified=%s\n",
				r.Slug, r.Title, tags, r.Category, r.Status, r.Created, r.Modified)
		}

		if searchLimit > 0 && len(results) == int(searchLimit) {
			fmt.Printf("(showing %d results, use --offset %d for more)\n", searchLimit, searchOffset+searchLimit)
		}
	},
}

func init() {
	searchCmd.Flags().StringSliceVarP(&searchTag, "tag", "t", nil, "Filter by tag (repeatable)")
	searchCmd.Flags().StringVarP(&searchCategory, "category", "c", "", "Filter by category")
	searchCmd.Flags().StringVarP(&searchStatus, "status", "s", "", "Filter by status")
	searchCmd.Flags().IntVarP(&searchLimit, "limit", "l", 0, "Limit N results")
	searchCmd.Flags().IntVarP(&searchOffset, "offset", "o", 0, "Skip N results (pagination with limit)")
}
