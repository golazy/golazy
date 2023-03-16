package lazyapp

import (
	"encoding/json"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golazy.dev/lazyaction"
	"golazy.dev/lazyassets"
	"golazy.dev/lazysupport"
)

var routesCmd = func(a *App) *cobra.Command {
	routesCmd := &cobra.Command{
		Use: "routes",
		Run: func(c *cobra.Command, args []string) {

			f := viper.GetString("format")
			if f != "text" && f != "json" {
				panic("Invalid format: " + f)
			}

			routes := a.Routes()
			vals := make([][]string, len(routes))

			for i, r := range routes {
				vals[i] = []string{r.Method, r.URL, r.Name}
			}

			var assets []lazyassets.Route

			if a.Assets != nil {
				assets = a.Assets.Routes()
			}

			if f == "json" {
				json.NewEncoder(os.Stdout).Encode(struct {
					Routes []lazyaction.Route `json:"routes,omitempty"`
					Assets []lazyassets.Route `json:"assets,omitempty"`
				}{routes, assets})

				return
			}

			t := lazysupport.Table{
				Header: []string{"Method", "Path", "Name"},
				Values: vals,
			}

			os.Stdout.Write([]byte(t.String()))
		},
	}
	routesCmd.Flags().StringP("format", "f", "text", "Output format (text, json)")
	viper.BindPFlag("format", routesCmd.Flags().Lookup("format"))
	viper.BindEnv("format")

	return routesCmd
}
