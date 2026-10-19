package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var indexDraft bool

var indexCmd = &cobra.Command{
	Use:   "index [--draft]",
	Short: "Emit a draft keyword index grouped by card category",
	Long: `Emits a draft of the 'index' card: every active card grouped by its
frontmatter category, with card titles as placeholder descriptions.
Navigation cards (index, *-index) are excluded.

Turn index maintenance from authoring into editing: fix the one-line
descriptions, move cards between categories, then save the result:

  zettel index > draft.md   # edit draft.md
  zettel write index < draft.md`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		draft, err := GetStore().DraftIndex()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(draft)
	},
}

func init() {
	// --draft is accepted (and default) so both `zettel index` and
	// `zettel index --draft` work; drafting is currently the only mode.
	indexCmd.Flags().BoolVar(&indexDraft, "draft", true, "Emit a draft index (the only mode; flag kept for explicitness)")
}
