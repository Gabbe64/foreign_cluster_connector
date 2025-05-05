package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var foreignclusters = &cobra.Command{
	Use:   "hello",
	Short: "Stampa un saluto",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Ciao mondo!")
	},
}

func init() {
	rootCmd.AddCommand(foreignclusters)
}