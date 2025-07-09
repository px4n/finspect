package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of finspect",
	Long:  `Print the version number and build information of finspect`,
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Printf("finspect version %s\n", Version)
		fmt.Printf("Built: %s\n", BuildTime)
	},
}
