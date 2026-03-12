package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func init() {
	manCmd.Flags().String("dir", "./manpages", "Output directory for man pages")
	rootCmd.AddCommand(manCmd)
}

var manCmd = &cobra.Command{
	Use:    "man",
	Short:  "Generate man pages",
	Long:   "Generate man pages for zd and write them to the specified directory.",
	Hidden: true,
	Args:   cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, _ := cmd.Flags().GetString("dir")

		header := &doc.GenManHeader{
			Title:   "ZD",
			Section: "1",
			Source:  fmt.Sprintf("zd %s", buildVersion),
			Manual:  "Zendesk CLI",
		}

		return doc.GenManTree(rootCmd, header, dir)
	},
}
