package cli

import (
	"fmt"
	"os"
	"path"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Used for flags.

	rootCmd = &cobra.Command{
		Use:   "lazy",
		Short: "Golazy companion tool",
		Long:  `lazy is a companion tool for the golazy framework.`,
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

var SubArgs = []string{}

func ExtractExtraArgs() {
	for i, arg := range os.Args {
		if arg == "--" {
			SubArgs = os.Args[i+1:]
			os.Args = os.Args[:i]
		}
	}
}

func init() {

	ExtractExtraArgs()

	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringP("author", "a", "YOUR NAME", "author name for copyright attribution")
	rootCmd.PersistentFlags().Bool("viper", true, "use Viper for configuration")
	viper.BindPFlag("author", rootCmd.PersistentFlags().Lookup("author"))
	viper.BindPFlag("useViper", rootCmd.PersistentFlags().Lookup("viper"))
	viper.SetDefault("author", "NAME HERE <EMAIL ADDRESS>")
	viper.SetDefault("license", "apache")

}

func initConfig() {
	// Find home directory.
	home, err := os.UserConfigDir()
	cobra.CheckErr(err)

	// Search config in home directory with name ".cobra" (without extension).
	viper.AddConfigPath(path.Join(home, "lazy"))
	viper.SetConfigType("yaml")
	viper.SetConfigName("config")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
