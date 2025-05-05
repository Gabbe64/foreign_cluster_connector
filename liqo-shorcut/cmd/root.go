package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "mycli",
	Short: "Un esempio di CLI con Cobra",
	Long:  `Questa è una CLI di esempio costruita usando Cobra in Go.`,
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
