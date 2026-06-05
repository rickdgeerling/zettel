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
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search cards by substring match and metadata filters",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		query := args[0]

		var catArg, statusArg *string
		if searchCategory != "" {
			catArg = &searchCategory
		}
		if searchStatus != "" {
			statusArg = &searchStatus
		}

		results, err := GetStore().SearchCards(query, searchTag, catArg, statusArg)
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
	},
}

func init() {
	searchCmd.Flags().StringSliceVarP(&searchTag, "tag", "t", nil, "Filter by tag (repeatable)")
	searchCmd.Flags().StringVarP(&searchCategory, "category", "c", "", "Filter by category")
	searchCmd.Flags().StringVarP(&searchStatus, "status", "s", "", "Filter by status")
}
