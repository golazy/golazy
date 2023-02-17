package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "new",
		Short: "Create a new lazy project",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("hola")
		},
	})
}

