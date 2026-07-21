package lazyapp

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"golazy.dev/lazyaddon"
	"golazy.dev/lazyassets"
	"golazy.dev/lazyroutes"
	"golazy.dev/lazytui/lazycli"
)

type cliRuntime struct {
	config Config
	app    *App
}

type cliRuntimeKey struct{}
type appContextKey struct{}

// FromContext returns the application initialized for a CLI invocation.
func FromContext(ctx context.Context) *App {
	app, _ := ctx.Value(appContextKey{}).(*App)
	return app
}

type appCLIController struct{}

func (*appCLIController) Before(ctx context.Context, _ struct{}) (context.Context, error) {
	runtime, _ := ctx.Value(cliRuntimeKey{}).(*cliRuntime)
	if runtime == nil {
		return ctx, fmt.Errorf("lazyapp: CLI runtime is unavailable")
	}
	if runtime.app == nil {
		runtime.app = New(runtime.config)
	}
	return context.WithValue(ctx, appContextKey{}, runtime.app), nil
}

func (*appCLIController) Run(ctx context.Context, _ struct{}) error {
	app := FromContext(ctx)
	if app == nil {
		return fmt.Errorf("lazyapp: application is unavailable")
	}
	return app.ListenAndServe()
}

func (*appCLIController) Cleanup(ctx context.Context, _ struct{}) error {
	app := FromContext(ctx)
	if app == nil {
		return nil
	}
	return app.Shutdown("application command finished")
}

type routesCLIController struct{}

type routesParams struct {
	JSON bool `lazycli:"help=Print routes as JSON Lines"`
}

func (*routesCLIController) Run(ctx context.Context, params routesParams) error {
	app := FromContext(ctx)
	if app == nil || app.Router == nil {
		return fmt.Errorf("lazyapp: application routes are unavailable")
	}
	if params.JSON {
		return writeRoutesJSONL(lazycli.Stdout(ctx), app.Router.Routes)
	}
	return writeRoutesTable(lazycli.Stdout(ctx), app.Router.Routes)
}

type assetsCLIController struct{}

func (*assetsCLIController) List(ctx context.Context, _ struct{}) error {
	app := FromContext(ctx)
	if app == nil || app.Assets == nil {
		return fmt.Errorf("lazyapp: application assets are unavailable")
	}
	assets := app.Assets.Manifest().Assets
	sort.SliceStable(assets, func(i, j int) bool { return assets[i].Path < assets[j].Path })
	for _, asset := range assets {
		fmt.Fprintln(lazycli.Stdout(ctx), asset.Path)
	}
	return nil
}

type unpackParams struct {
	Dir  string `lazycli:"default=public-build,help=Output directory"`
	Mode string `lazycli:"default=both,enum=both|logical|permanent,help=Asset path forms to write"`
}

func (*assetsCLIController) Unpack(ctx context.Context, params unpackParams) error {
	app := FromContext(ctx)
	if app == nil || app.Assets == nil {
		return fmt.Errorf("lazyapp: application assets are unavailable")
	}
	mode := lazyassets.UnpackBoth
	switch params.Mode {
	case "logical":
		mode = lazyassets.UnpackLogical
	case "permanent":
		mode = lazyassets.UnpackPermanent
	}
	return app.Assets.Unpack(params.Dir, lazyassets.WithUnpackMode(mode))
}

func defaultCLI(s lazycli.Scope) {
	s.Root(lazycli.Command{
		Controller:   lazycli.Controller[appCLIController]{},
		Autocomplete: true,
	})
	s.Define("routes", lazycli.Command{
		Help:       "Print application HTTP routes",
		Controller: lazycli.Controller[routesCLIController]{},
	})
	s.Group("assets", lazycli.Command{
		Help:       "Inspect and unpack application assets",
		Controller: lazycli.Controller[assetsCLIController]{},
	}, func(s lazycli.Scope) {
		s.Define("list", lazycli.Command{Help: "List packaged application assets"})
		s.Define("unpack", lazycli.Command{Help: "Unpack packaged application assets"})
	})
}

// Run executes the application command router and returns a process-style exit
// code. With no arguments, the framework root controller starts ListenAndServe.
func Run(config Config) int {
	return Execute(context.Background(), config, lazycli.ExecuteConfig{
		Args: os.Args[1:], Stdin: os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr, Name: os.Args[0],
	})
}

// Execute is the stream-injectable form of Run.
func Execute(ctx context.Context, config Config, execution lazycli.ExecuteConfig) int {
	if execution.Stdout == nil {
		execution.Stdout = os.Stdout
	}
	if execution.Stderr == nil {
		execution.Stderr = os.Stderr
	}
	addons, err := lazyaddon.Resolve(config.Addons)
	if err != nil {
		fmt.Fprintf(execution.Stderr, "lazyapp: resolve add-ons: %v\n", err)
		return 1
	}
	draw := func(s lazycli.Scope) {
		s.Merge(defaultCLI)
		runAddonHook(addons, CLIHook, &CLIEvent{Scope: s, Addons: addons})
		s.Merge(config.CLI)
	}
	ctx = context.WithValue(ctx, cliRuntimeKey{}, &cliRuntime{config: config})
	return lazycli.Execute(ctx, draw, execution)
}

func writeRoutesTable(w interface{ Write([]byte) (int, error) }, routes lazyroutes.RouteTable) error {
	rows := [][]string{{"Name", "Method", "Path", "Controller#Action"}}
	for _, route := range routes {
		target := strings.Trim(route.Controller, "/")
		if route.Action != "" {
			if target != "" {
				target += "#"
			}
			target += route.Action
		}
		rows = append(rows, []string{route.Name, route.Method, route.Path, target})
	}
	widths := make([]int, len(rows[0]))
	for _, row := range rows {
		for index, value := range row {
			if len(value) > widths[index] {
				widths[index] = len(value)
			}
		}
	}
	for _, row := range rows {
		var line strings.Builder
		for index, value := range row {
			if index > 0 {
				line.WriteString("  ")
			}
			fmt.Fprintf(&line, "%-*s", widths[index], value)
		}
		if _, err := fmt.Fprintln(w, strings.TrimRight(line.String(), " ")); err != nil {
			return err
		}
	}
	return nil
}
