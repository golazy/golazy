package lazyapp

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"strings"

	"golazy.dev/lazyassets"
	"golazy.dev/lazycache"
	"golazy.dev/lazycache/inmemorycache"
	"golazy.dev/lazycontroller"
	"golazy.dev/lazycontrolplane"
	"golazy.dev/lazydeps"
	"golazy.dev/lazydispatch"
	"golazy.dev/lazydispatch/middlewares"
	"golazy.dev/lazyforms"
	"golazy.dev/lazyjobs"
	"golazy.dev/lazyjobs/inmemoryjobs"
	"golazy.dev/lazyroutes"
	"golazy.dev/lazyseo"
	"golazy.dev/lazysession"
	"golazy.dev/lazytelemetry"
	"golazy.dev/lazytelemetry/lazymetrics"
	"golazy.dev/lazyturbo"
	_ "golazy.dev/lazyview/gotmpl"
)

type Helpers []map[string]any

type Config struct {
	Name              string
	Drawer            func(*lazyroutes.Scope)
	Public            func() (fs.FS, error)
	Views             func() (fs.FS, error)
	Dependencies      func(*lazydeps.Scope) error
	Helpers           Helpers
	SEO               func(context.Context) []lazyseo.Option
	Assets            []lazyassets.Source
	AssetOptions      []lazyassets.Option
	Cache             lazycache.Options
	Robots            RobotsConfig
	Sitemap           SitemapConfig
	Sessions          lazysession.Config
	Jobs              lazyjobs.Config
	ControlPlane      lazycontrolplane.Builder
	Middlewares       []lazydispatch.Middleware
	ForceDetailErrors bool
}

type App struct {
	Name         string
	Context      context.Context
	Dispatcher   *lazydispatch.Dispatcher
	Router       *lazyroutes.Scope
	Assets       *lazyassets.Registry
	Cache        *lazycache.Cache
	Sessions     *lazysession.Manager
	Jobs         *lazyjobs.JobRunner
	ControlPlane *lazycontrolplane.ControlPlane
	Dependencies *lazydeps.Scope
}

type assetsMiddleware struct {
	registry *lazyassets.Registry
}

func (assetsMiddleware) MiddlewareName() string {
	return "lazyassets.Registry"
}

func (middleware assetsMiddleware) Handler(next http.Handler) http.Handler {
	if middleware.registry == nil {
		return next
	}
	return middleware.registry.Handler(next)
}

var afterDraw = func(*lazyroutes.Scope) {}

func MustSub(fsys fs.FS, dir string) func() (fs.FS, error) {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(fmt.Errorf("open %s: %w", dir, err))
	}
	return func() (fs.FS, error) {
		return sub, nil
	}
}

func New(config Config) *App {
	ctx := context.Background()
	telemetryConfig := lazytelemetry.MustLoadConfig()
	telemetryRegistry := lazymetrics.NewRegistry()
	var controlPlane *lazycontrolplane.ControlPlane
	if config.ControlPlane != nil {
		controlPlane = config.ControlPlane.BuildControlPlane()
		if controlPlane == nil {
			panic("lazyapp: control plane builder returned nil")
		}
	}
	cacheOptions := config.Cache
	if cacheOptions.Backend == nil {
		backend, err := inmemorycache.New(inmemorycache.Options{})
		if err != nil {
			panic(fmt.Errorf("initialize cache backend: %w", err))
		}
		cacheOptions.Backend = backend
	}
	cache, err := lazycache.New(cacheOptions)
	if err != nil {
		panic(fmt.Errorf("initialize cache: %w", err))
	}
	ctx = lazycache.WithCache(ctx, cache)

	var sessions *lazysession.Manager
	if config.Sessions.Enabled() {
		sessionConfig := config.Sessions
		if sessionConfig.Name == "" && config.Name != "" {
			sessionConfig.Name = derivedSessionName(config.Name)
		}
		var err error
		sessions, err = lazysession.NewManager(sessionConfig)
		if err != nil {
			panic(fmt.Errorf("initialize sessions: %w", err))
		}
		ctx = lazysession.WithManager(ctx, sessions)
	}

	defaultViews, err := lazycontroller.DefaultViews()
	if err != nil {
		panic(err)
	}
	views := defaultViews
	if config.Views != nil {
		configuredViews, err := openConfiguredViews(config.Views)
		if err != nil {
			panic(fmt.Errorf("open views: %w", err))
		}
		views = overlayViewFS(configuredViews, defaultViews)
	}
	renderer, err := lazycontroller.NewRenderer(views)
	if err != nil {
		panic(fmt.Errorf("initialize renderer: %w", err))
	}
	ctx = lazycontroller.WithRenderer(ctx, renderer)
	if config.ForceDetailErrors {
		ctx = lazycontroller.WithDetailErrors(ctx)
	}
	ctx = lazyDevContext(ctx)

	assetOptions := append([]lazyassets.Option{}, config.AssetOptions...)
	assetOptions = append(assetOptions, lazyDevAssetOptions()...)
	assets := lazyassets.New(assetOptions...)
	if config.Public != nil {
		public, err := openConfiguredPublic(config.Public)
		if err != nil {
			panic(fmt.Errorf("open public files: %w", err))
		}
		ctx = lazycontroller.WithErrorPages(ctx, public)
		if err := assets.AddFS(public); err != nil {
			panic(fmt.Errorf("register public assets: %w", err))
		}
	}
	for _, source := range config.Assets {
		if source == nil {
			panic(fmt.Errorf("register generated assets: asset source is nil"))
		}
		if err := source.Assets(assets); err != nil {
			panic(fmt.Errorf("register generated assets: %w", err))
		}
	}

	dependencies := lazydeps.New(ctx)
	if config.Dependencies != nil {
		if err := config.Dependencies(dependencies); err != nil {
			panic(fmt.Errorf("initialize dependencies: %w", err))
		}
		ctx = dependencies.Context()
	}

	var jobs *lazyjobs.JobRunner
	if jobsConfigured(config.Jobs) {
		jobsConfig := config.Jobs
		if jobsConfig.Backend == nil {
			jobsConfig.Backend = inmemoryjobs.New()
		}
		var err error
		jobs, err = lazyjobs.New(jobsConfig)
		if err != nil {
			panic(fmt.Errorf("initialize jobs: %w", err))
		}
		ctx = lazyjobs.WithRunner(ctx, jobs)
		dependencies.SetContext(ctx)
		jobs.Start(ctx)
	}

	var seo []lazyseo.Option
	if config.SEO != nil {
		seo = config.SEO(ctx)
	}

	router := lazyroutes.New(ctx)
	ctx = lazycontroller.WithPathFor(ctx, router.PathFor)
	router.Context = ctx
	if config.Drawer != nil {
		config.Drawer(router)
	}
	afterDraw(router)
	renderer.AddHelpers(router.RegisterHelpers())
	renderer.AddHelpers(assets.Helpers())
	renderer.AddHelpers(lazyforms.Helpers(router))
	renderer.AddHelpers(lazyseo.Helpers(seo...))
	renderer.AddHelpers(lazyturbo.Helpers())
	renderer.AddHelpers(cacheHelpers())
	for _, helpers := range config.Helpers {
		renderer.AddHelpers(helpers)
	}
	controlPlane = jobsControlPlane(controlPlane, jobs)
	controlPlane = lazyDevControlPlane(controlPlane, renderer, router, assets, cache, jobs)
	controlPlane = telemetryControlPlane(controlPlane, telemetryConfig, telemetryRegistry, cache)
	if err := renderer.Cache(); err != nil {
		panic(fmt.Errorf("cache views: %w", err))
	}

	dispatcher := lazydispatch.NewDispatcher()
	if telemetryMiddlewareEnabled(telemetryConfig) {
		dispatcher.Use(lazytelemetry.MiddlewareFromConfig(telemetryConfig, lazytelemetry.WithMetricsRegistry(telemetryRegistry)))
	}
	dispatcher.Use(lazydispatch.RouteOnly(
		router,
		middlewares.MethodOverride(),
		lazydispatch.ResponseBuffer(),
		lazydispatch.MiddlewareFunc(lazycontroller.ErrorHandler(ctx)),
		lazydispatch.ETag(),
	))
	if sessions != nil {
		dispatcher.Use(sessions)
	}
	for _, middleware := range config.Middlewares {
		dispatcher.Use(middleware)
	}
	metadata, err := newMetadataFiles(config.Robots, config.Sitemap)
	if err != nil {
		panic(fmt.Errorf("initialize metadata files: %w", err))
	}
	dispatcher.Use(metadata)
	dispatcher.Use(lazydispatch.Router(router))
	if !assets.Empty() {
		dispatcher.Use(assetsMiddleware{registry: assets})
	}

	return &App{
		Name:         config.Name,
		Context:      ctx,
		Dispatcher:   dispatcher,
		Router:       router,
		Assets:       assets,
		Cache:        cache,
		Sessions:     sessions,
		Jobs:         jobs,
		ControlPlane: controlPlane,
		Dependencies: dependencies,
	}
}

func jobsConfigured(config lazyjobs.Config) bool {
	return config.Backend != nil ||
		config.Define != nil ||
		config.Workers != 0 ||
		config.PollInterval != 0 ||
		len(config.Queues) > 0
}

func (app *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if controlPlane := app.controlPlaneInServeHTTP(); controlPlane != nil && controlPlane.HandlesPath(r.URL.Path) {
		controlPlane.ServeHTTP(w, r)
		return
	}
	app.Dispatcher.ServeHTTP(w, r)
}

// ListenAndServe starts the app server on ADDR, PORT, or 127.0.0.1:3000.
//
// It installs app.Context as the server base context, so every request context
// includes the dependencies initialized by New. When using a custom http.Server,
// set BaseContext to return app.Context.
func (app *App) ListenAndServe() error {
	appAddr := listenAddr()
	controlAddr, controlAddrSet := controlPlaneListenAddr()
	appHandler, controlHandler := app.handlersForListen(appAddr, controlAddr, controlAddrSet)
	appServer := app.newServer(appAddr, appHandler)
	if controlHandler == nil {
		return listenAndServe(appServer)
	}

	controlServer := app.newServer(controlAddr, controlHandler)
	return listenAndServeBoth(appServer, controlServer)
}

func sameListenAddr(left, right string) bool {
	left = normalizeListenAddr(left)
	right = normalizeListenAddr(right)
	if left == right {
		return true
	}

	leftHost, leftPort, leftOK := splitListenAddr(left)
	rightHost, rightPort, rightOK := splitListenAddr(right)
	if !leftOK || !rightOK || leftPort != rightPort {
		return false
	}
	return listenHostsOverlap(leftHost, rightHost)
}

func splitListenAddr(addr string) (host string, port string, ok bool) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", "", false
	}
	return strings.ToLower(host), port, true
}

func listenHostsOverlap(left, right string) bool {
	if left == right {
		return true
	}
	if isWildcardListenHost(left) || isWildcardListenHost(right) {
		return true
	}
	return isLocalListenHost(left) && isLocalListenHost(right)
}

func isWildcardListenHost(host string) bool {
	return host == "" || host == "0.0.0.0" || host == "::"
}

func isLocalListenHost(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func (app *App) controlPlaneForListen(controlAddrSet bool) *lazycontrolplane.ControlPlane {
	if !controlAddrSet {
		return app.controlPlaneWithoutListenAddress()
	}
	if app.ControlPlane != nil {
		return app.ControlPlane
	}
	return lazycontrolplane.New(lazycontrolplane.Config{})
}

func (app *App) handlersForListen(appAddr string, controlAddr string, controlAddrSet bool) (http.Handler, http.Handler) {
	controlPlane := app.controlPlaneForListen(controlAddrSet)
	appHandler := http.Handler(app.Dispatcher)
	if controlPlane == nil {
		return appHandler, nil
	}
	if controlAddrSet && sameListenAddr(appAddr, controlAddr) {
		return controlPlane.Handler(appHandler), nil
	}
	if controlAddrSet {
		controlPlane.EnablePprof()
		return appHandler, controlPlane.StandaloneHandler()
	}
	return appHandler, controlPlane
}

func (app *App) newServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:    addr,
		Handler: handler,
		BaseContext: func(_ net.Listener) context.Context {
			return app.Context
		},
	}
}

func listenAndServe(server *http.Server) error {
	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func listenAndServeBoth(appServer *http.Server, controlServer *http.Server) error {
	errs := make(chan error, 2)
	go func() {
		errs <- listenAndServe(controlServer)
	}()
	go func() {
		errs <- listenAndServe(appServer)
	}()

	err := <-errs
	_ = appServer.Close()
	_ = controlServer.Close()
	if secondErr := <-errs; err == nil {
		err = secondErr
	}
	return err
}

func derivedSessionName(appName string) string {
	appName = strings.TrimSpace(appName)
	if index := strings.LastIndex(appName, "/"); index >= 0 {
		appName = appName[index+1:]
	}

	var builder strings.Builder
	lastUnderscore := false
	for _, r := range appName {
		if isSessionNameRune(r) {
			builder.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			builder.WriteByte('_')
			lastUnderscore = true
		}
	}

	name := strings.Trim(builder.String(), "_")
	if name == "" {
		return ""
	}
	return name + "_session"
}

func isSessionNameRune(r rune) bool {
	return r == '.' ||
		r == '-' ||
		r == '_' ||
		('0' <= r && r <= '9') ||
		('A' <= r && r <= 'Z') ||
		('a' <= r && r <= 'z')
}
