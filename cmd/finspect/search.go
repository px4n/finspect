package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search for files",
	Long:  `Search for files across all mounted sources. (Not implemented in Phase 1)`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Println("Search functionality will be implemented in Phase 2")
		fmt.Println("This will include:")
		fmt.Println("- Full-text search across file contents")
		fmt.Println("- Metadata-based search")
		fmt.Println("- Query language support")
	},
}
