package cmd

import (
	"fmt"
	"os"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "liqo-shorcut",
	Short: "a cli command used to create a shortcut between two foreign clusters",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Ciao dal comando root!")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
