package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

var (
	appendSection string
	appendText    string
)

var appendCmd = &cobra.Command{
	// Text is taken via --text or stdin (not a positional) because append
	// lines usually start with "- " and would otherwise be parsed as flags.
	Use:   "append <slug>",
	Short: "Append a line to a card section (prose-safe, creates the section if absent)",
	Long: `Appends a line of text to the card's "## <section>" section without
touching any existing prose — the safe primitive for append-only edits like
adding entries to a ## Links section. The section is created at the end of
the body when absent. modified is bumped.

Text comes from the --text flag or stdin, mirroring zettel write:

  zettel append my-card --text "- [[other]]: related"
  echo "- [[other]]: related" | zettel append my-card`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		slug := args[0]

		text := appendText
		if text == "" && stdinPiped() {
			data, _ := io.ReadAll(os.Stdin)
			text = string(data)
		}
		if text == "" {
			fmt.Fprintf(os.Stderr, "Error: provide the line to append via --text flag or stdin\n")
			os.Exit(1)
		}

		if err := GetStore().AppendToSection(slug, appendSection, text); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Appended to %q section of %q\n", appendSection, slug)
	},
}

func init() {
	appendCmd.Flags().StringVar(&appendSection, "section", "Links", "Section heading to append to (created if absent)")
	appendCmd.Flags().StringVar(&appendText, "text", "", "Line to append (reads stdin when omitted)")
}
