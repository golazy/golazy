
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "serve",
		Short: "Start development server",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("hola")
		},
	})
}

