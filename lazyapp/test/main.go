package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func main() {
	viper.SetConfigFile(".env")
	viper.ReadInConfig()

	command := &cobra.Command{
		Run: func(c *cobra.Command, args []string) {
			fmt.Println(viper.GetString("listen"))
		},
	}

	command.PersistentFlags().StringP("listen", "l", "localhost:2000", "Listen address")
	command.PersistentFlags().StringP("port", "", "", "http port. Ignored if listen is set")
	viper.BindPFlag("listen", command.PersistentFlags().Lookup("listen"))
	viper.BindPFlag("port", command.PersistentFlags().Lookup("listen"))
	viper.BindEnv("listen")
	viper.BindEnv("port")

	viper.SetDefault("listen", "localhost:2000")

	command.Execute()

}
