package lazyapp

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"golazy.dev/lazyassets"
	"golazy.dev/lazyauth"
	"golazy.dev/lazyauth/memoryauth"
	"golazy.dev/lazybuildinfo"
	"golazy.dev/lazycache"
	"golazy.dev/lazycache/inmemorycache"
	"golazy.dev/lazycontroller"
	"golazy.dev/lazycontrolplane"
	"golazy.dev/lazydeps"
	"golazy.dev/lazydispatch"
	"golazy.dev/lazydispatch/middlewares"
	"golazy.dev/lazyfiles"
	"golazy.dev/lazyforms"
	"golazy.dev/lazyjobs"
	"golazy.dev/lazyjobs/inmemoryjobs"
	"golazy.dev/lazymcp"
	"golazy.dev/lazymedia"
	"golazy.dev/lazymigrate"
	"golazy.dev/lazyoauth"
	"golazy.dev/lazypwa"
	"golazy.dev/lazyroutes"
	"golazy.dev/lazyseo"
	"golazy.dev/lazysession"
	"golazy.dev/lazystorage"
	"golazy.dev/lazyturbo"
	_ "golazy.dev/lazyview/gotmpl"
	"golazy.dev/lazyworkers"
)

type Helpers []map[string]any

const appShutdownTimeout = 10 * time.Second

// WorkersConfig registers browser workers with the dependency-initialized app
// context.
type WorkersConfig func(context.Context, *lazyworkers.Registry) error

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
	Storages          map[string]lazystorage.Storage
	Files             *lazyfiles.Files
	Media             *lazymedia.Media
	Cache             lazycache.Options
	Robots            RobotsConfig
	Sitemap           SitemapConfig
	Sessions          lazysession.Config
	Migrations        MigrationsConfig
	Jobs              JobsConfig
	Auth              AuthConfig
	OAuth             OAuthConfig
	MCP               MCPConfig
	MCPOptions        lazymcp.Options
	Workers           WorkersConfig
	PWA               lazypwa.Config
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
	Storages     map[string]lazystorage.Storage
	Files        *lazyfiles.Files
	Media        *lazymedia.Media
	Cache        *lazycache.Cache
	Sessions     *lazysession.Manager
	Migrations   lazymigrate.Databases
	Jobs         *lazyjobs.JobRunner
	Auth         lazyauth.Config
	OAuth        *lazyoauth.Server
	MCP          *lazymcp.Scope
	Workers      *lazyworkers.Registry
	PWA          *lazypwa.App
	ControlPlane *lazycontrolplane.ControlPlane
	Dependencies *lazydeps.Scope
	runtime      *runtimeState
	earlyControl *migrationControlPlane
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

const defaultCacheMaxSizeBytes int64 = 50 * 1024 * 1024

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
	ctx = lazycache.WithBuildVersion(ctx, appBuildVersion())
	runtime := newRuntimeState()
	migrationMode, err := configuredMigrationMode()
	if err != nil {
		panic(err)
	}
	var controlPlane *lazycontrolplane.ControlPlane
	if config.ControlPlane != nil {
		controlPlane = config.ControlPlane.BuildControlPlane()
		if controlPlane == nil {
			panic("lazyapp: control plane builder returned nil")
		}
	}
	controlPlane, migrationReadiness, migrationControl, err := prepareMigrationControlPlane(migrationMode, controlPlane)
	if err != nil {
		panic(err)
	}
	returned := false
	defer func() {
		if !returned && migrationControl != nil {
			_ = migrationControl.Close()
		}
	}()
	cacheOptions := config.Cache
	if cacheOptions.Backend == nil {
		backend, err := inmemorycache.New(inmemorycache.Options{MaxSizeBytes: defaultCacheMaxSizeBytesFromEnvironment()})
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
	configuredAuth := lazyauth.Config{Authenticator: memoryauth.FromEnvironment()}
	ctx = lazyauth.WithConfig(ctx, configuredAuth)

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
	ctx = lazyassets.WithRegistry(ctx, assets)

	dependencies := lazydeps.New(ctx)
	telemetry, err := initializeTelemetry(dependencies)
	if err != nil {
		panic(fmt.Errorf("initialize telemetry: %w", err))
	}
	ctx = dependencies.Context()
	if config.Dependencies != nil {
		if err := config.Dependencies(dependencies); err != nil {
			panic(fmt.Errorf("initialize dependencies: %w", err))
		}
		ctx = dependencies.Context()
	}
	ctx = lazyauth.WithConfig(ctx, configuredAuth)
	dependencies.SetContext(ctx)

	if config.Auth != nil {
		auth, err := config.Auth(ctx)
		if err != nil {
			panic(fmt.Errorf("initialize auth: %w", err))
		}
		configuredAuth = auth
		ctx = lazyauth.WithConfig(ctx, configuredAuth)
		dependencies.SetContext(ctx)
	}

	var oauthServer *lazyoauth.Server
	if config.OAuth != nil {
		oauthConfig, err := config.OAuth(ctx)
		if err != nil {
			panic(fmt.Errorf("initialize oauth: %w", err))
		}
		if oauthConfig.Auth.Authenticator == nil {
			oauthConfig.Auth = configuredAuth
		}
		oauthServer, err = lazyoauth.New(oauthConfig)
		if err != nil {
			panic(fmt.Errorf("initialize oauth: %w", err))
		}
	}

	var migrations lazymigrate.Databases
	if config.Migrations != nil {
		configuredMigrations, err := config.Migrations(ctx)
		if err != nil {
			panic(fmt.Errorf("initialize migrations: %w", err))
		}
		migrations = configuredMigrations
	}
	if migrationMode != migrationModeOff {
		if err := applyConfiguredMigrations(ctx, migrations); err != nil {
			panic(fmt.Errorf("run migrations: %w", err))
		}
		migrationReadiness.Done()
		if migrationMode == migrationModeUp {
			if migrationControl != nil {
				if err := migrationControl.Close(); err != nil {
					panic(err)
				}
				migrationControl = nil
			}
			exitAfterMigrate(0)
			panic("lazyapp: exit after migrations returned")
		}
	}

	var jobs *lazyjobs.JobRunner
	if config.Jobs != nil {
		jobsConfig, err := config.Jobs(ctx)
		if err != nil {
			panic(fmt.Errorf("initialize jobs: %w", err))
		}
		if jobsConfig.Backend == nil {
			jobsConfig.Backend = inmemoryjobs.New()
		}
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

	workers := lazyworkers.New()
	if config.Workers != nil {
		if err := config.Workers(ctx, workers); err != nil {
			panic(fmt.Errorf("initialize workers: %w", err))
		}
	}

	var pwa *lazypwa.App
	if config.PWA.IsEnabled() {
		pwa, err = lazypwa.New(config.PWA,
			lazypwa.WithAppName(config.Name),
			lazypwa.WithVersion(lazycache.BuildVersionFromContext(ctx)),
			lazypwa.WithAssets(assets),
			lazypwa.WithWorkers(workers),
		)
		if err != nil {
			panic(fmt.Errorf("initialize PWA: %w", err))
		}
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
	renderer.AddHelpers(workers.Helpers())
	if pwa != nil && pwa.Enabled() {
		renderer.AddHelpers(pwa.Helpers())
	}
	renderer.AddHelpers(lazyforms.Helpers(router))
	renderer.AddHelpers(lazyseo.Helpers(seo...))
	renderer.AddHelpers(lazyturbo.Helpers())
	renderer.AddHelpers(cacheHelpers())
	for _, helpers := range config.Helpers {
		renderer.AddHelpers(helpers)
	}
	var mcpScope *lazymcp.Scope
	if config.MCP != nil {
		options := config.MCPOptions
		if options.Views == nil {
			options.Views = mcpViews{views: renderer}
		}
		mcpScope = lazymcp.NewScope(options)
		if err := config.MCP(ctx, mcpScope); err != nil {
			panic(fmt.Errorf("initialize mcp: %w", err))
		}
	}
	media := newMediaServices(config)
	controlPlane = jobsControlPlane(controlPlane, jobs)
	controlPlane = lazyDevControlPlane(controlPlane, renderer, router, assets, cache, dependencies, jobs, workers, pwa, runtime, media)
	controlPlane = telemetryControlPlane(controlPlane, telemetry, cache)
	if err := renderer.Cache(); err != nil {
		panic(fmt.Errorf("cache views: %w", err))
	}

	dispatcher := lazydispatch.NewDispatcher()
	if telemetryMiddlewareEnabled(telemetry.Config()) {
		dispatcher.Use(telemetry.Middleware())
	}
	dispatcher.Use(lazydispatch.RouteOnly(
		router,
		middlewares.DynamicRoute(ctx),
	))
	if !workers.Empty() {
		dispatcher.Use(workers)
	}
	if pwa != nil && pwa.Enabled() {
		dispatcher.Use(pwa)
	}
	if sessions != nil {
		dispatcher.Use(sessions)
	}
	if mcpScope != nil && !mcpScope.Empty() {
		dispatcher.Use(mcpMiddleware{oauth: oauthServer, mcp: mcpScope})
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
	if migrationControl != nil {
		migrationControl.SetBaseContext(ctx)
	}

	app := &App{
		Name:         config.Name,
		Context:      ctx,
		Dispatcher:   dispatcher,
		Router:       router,
		Assets:       assets,
		Storages:     media.Storages,
		Files:        media.Files,
		Media:        media.Media,
		Cache:        cache,
		Sessions:     sessions,
		Migrations:   migrations,
		Jobs:         jobs,
		Auth:         configuredAuth,
		OAuth:        oauthServer,
		MCP:          mcpScope,
		Workers:      workers,
		PWA:          pwa,
		ControlPlane: controlPlane,
		Dependencies: dependencies,
		runtime:      runtime,
		earlyControl: migrationControl,
	}
	returned = true
	return app
}

type mediaServices struct {
	Storages       map[string]lazystorage.Storage
	DefaultStorage string
	Files          *lazyfiles.Files
	Media          *lazymedia.Media
}

func newMediaServices(config Config) mediaServices {
	storages := map[string]lazystorage.Storage{}
	for name, storage := range config.Storages {
		if name == "" || storage == nil {
			continue
		}
		storages[name] = storage
	}
	if config.Files != nil {
		for name, storage := range config.Files.Storages {
			if name == "" || storage == nil {
				continue
			}
			if _, exists := storages[name]; !exists {
				storages[name] = storage
			}
		}
	}
	return mediaServices{
		Storages:       storages,
		DefaultStorage: defaultMediaStorage(config),
		Files:          config.Files,
		Media:          config.Media,
	}
}

func defaultMediaStorage(config Config) string {
	if config.Files != nil && config.Files.DefaultStorage != "" {
		return config.Files.DefaultStorage
	}
	names := make([]string, 0, len(config.Storages))
	for name := range config.Storages {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return ""
	}
	return names[0]
}

func appBuildVersion() string {
	return lazybuildinfo.Version()
}

func (app *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if controlPlane := app.controlPlaneInServeHTTP(); controlPlane != nil && controlPlane.HandlesPath(r.URL.Path) {
		controlPlane.ServeHTTP(w, r)
		return
	}
	app.handler().ServeHTTP(w, r)
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
	appServer := app.newServer(appAddr, appHandler, true)
	if controlHandler == nil {
		return app.listenAndShutdown(appServer)
	}
	if app.hasEarlyControlPlane(controlAddr) && !sameListenAddr(appAddr, controlAddr) {
		return app.listenAndServeAppWithEarlyControl(appServer)
	}

	controlServer := app.newServer(controlAddr, controlHandler, false)
	return app.listenAndShutdown(appServer, controlServer)
}

func (app *App) hasEarlyControlPlane(controlAddr string) bool {
	if app == nil || app.earlyControl == nil {
		return false
	}
	return app.earlyControl.ActiveOn(controlAddr)
}

func (app *App) listenAndServeAppWithEarlyControl(appServer *http.Server) error {
	err := listenAndServeServers(appServer)
	if closeErr := app.closeEarlyControl(); closeErr != nil {
		err = errors.Join(err, closeErr)
	}
	if shutdownErr := app.shutdownDependencies("listen-and-serve stopped"); shutdownErr != nil {
		err = errors.Join(err, shutdownErr)
	}
	return err
}

func (app *App) listenAndShutdown(servers ...*http.Server) error {
	err := listenAndServeServers(servers...)
	if shutdownErr := app.shutdownDependencies("listen-and-serve stopped"); shutdownErr != nil {
		err = errors.Join(err, shutdownErr)
	}
	return err
}

func (app *App) shutdownDependencies(reason string) error {
	if app == nil || app.Dependencies == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), appShutdownTimeout)
	defer cancel()
	return app.Dependencies.Shutdown(ctx, reason)
}

func (app *App) closeEarlyControl() error {
	if app == nil || app.earlyControl == nil {
		return nil
	}
	control := app.earlyControl
	app.earlyControl = nil
	return control.Close()
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
	appHandler := app.handler()
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

func (app *App) handler() http.Handler {
	if app == nil || app.runtime == nil {
		return app.Dispatcher
	}
	return app.runtime.Handler(app.Dispatcher)
}

func (app *App) newServer(addr string, handler http.Handler, trackConnections bool) *http.Server {
	server := &http.Server{
		Addr:    addr,
		Handler: handler,
		BaseContext: func(_ net.Listener) context.Context {
			return app.Context
		},
	}
	if trackConnections && app != nil && app.runtime != nil {
		server.ConnState = app.runtime.ConnState
	}
	return server
}

func listenAndServe(server *http.Server) error {
	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func listenAndServeBoth(appServer *http.Server, controlServer *http.Server) error {
	return listenAndServeServers(appServer, controlServer)
}

func listenAndServeServers(servers ...*http.Server) error {
	if len(servers) == 0 {
		return nil
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errs := make(chan error, len(servers))
	for _, server := range servers {
		go func(server *http.Server) {
			errs <- listenAndServe(server)
		}(server)
	}

	select {
	case <-ctx.Done():
		stop()
		err := shutdownServers(servers...)
		for range servers {
			if serveErr := <-errs; serveErr != nil {
				err = errors.Join(err, serveErr)
			}
		}
		return err
	case err := <-errs:
		for _, server := range servers {
			_ = server.Close()
		}
		for index := 1; index < len(servers); index++ {
			if serveErr := <-errs; serveErr != nil {
				err = errors.Join(err, serveErr)
			}
		}
		return err
	}
}

func shutdownServers(servers ...*http.Server) error {
	ctx, cancel := context.WithTimeout(context.Background(), appShutdownTimeout)
	defer cancel()

	var err error
	for _, server := range servers {
		if server == nil {
			continue
		}
		if shutdownErr := server.Shutdown(ctx); shutdownErr != nil && !errors.Is(shutdownErr, http.ErrServerClosed) {
			err = errors.Join(err, shutdownErr)
		}
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

func defaultCacheMaxSizeBytesFromEnvironment() int64 {
	if environment.LazyappCacheSize > 0 {
		return int64(environment.LazyappCacheSize)
	}
	return defaultCacheMaxSizeBytes
}
