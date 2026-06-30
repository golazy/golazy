package lazypwa

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"path"
	"sort"
	"strings"

	"golazy.dev/lazyassets"
	"golazy.dev/lazybuildinfo"
	"golazy.dev/lazyview"
	"golazy.dev/lazyworkers"
)

const (
	defaultManifestPath      = "/manifest.webmanifest"
	defaultServiceWorkerName = "pwa"
	defaultServiceWorkerPath = "/service-worker.js"
	defaultCacheManifestPath = "/pwa-cache-manifest.json"
	defaultClientAssetPath   = "/assets/lazypwa.js"
)

// Config describes an app's progressive web app behavior.
type Config struct {
	Installable       bool
	Version           string
	ManifestPath      string
	ServiceWorkerName string
	ServiceWorkerPath string
	CacheManifestPath string
	ClientAssetPath   string
	Manifest          ManifestConfig
	Offline           OfflineConfig
	Push              PushConfig
}

// ManifestConfig describes the generated web app manifest.
type ManifestConfig struct {
	Name            string
	ShortName       string
	Description     string
	StartURL        string
	Scope           string
	Display         string
	ThemeColor      string
	BackgroundColor string
	Orientation     string
	Categories      []string
	Icons           []Icon
}

// Icon describes one web app manifest icon.
type Icon struct {
	Src     string `json:"src"`
	Sizes   string `json:"sizes,omitempty"`
	Type    string `json:"type,omitempty"`
	Purpose string `json:"purpose,omitempty"`
}

// OfflineConfig controls opt-in offline caching.
type OfflineConfig struct {
	Enabled       bool
	URLs          []string
	Assets        []string
	Sources       []OfflineSource
	FallbackURL   string
	IncludeAssets bool
}

// OfflineSource supplies URLs for the offline cache manifest.
type OfflineSource interface {
	OfflineURLs() ([]string, error)
}

// OfflineSourceFunc adapts a function into an OfflineSource.
type OfflineSourceFunc func() ([]string, error)

// OfflineURLs calls fn.
func (fn OfflineSourceFunc) OfflineURLs() ([]string, error) {
	return fn()
}

// PushConfig describes browser push subscription support.
type PushConfig struct {
	Enabled          bool
	ApplicationKey   string
	SubscriptionPath string
	Store            SubscriptionStore
	Sender           PushSender
}

// PushSubscription is the browser PushSubscription shape applications store.
type PushSubscription struct {
	Endpoint       string         `json:"endpoint"`
	ExpirationTime *int64         `json:"expirationTime,omitempty"`
	Keys           map[string]any `json:"keys,omitempty"`
	UserKey        string         `json:"user_key,omitempty"`
}

// PushMessage is an application notification payload.
type PushMessage struct {
	Title string         `json:"title,omitempty"`
	Body  string         `json:"body,omitempty"`
	Data  map[string]any `json:"data,omitempty"`
}

// SubscriptionStore stores browser push subscriptions.
type SubscriptionStore interface {
	SavePushSubscription(context.Context, PushSubscription) error
	DeletePushSubscription(context.Context, string) error
}

// PushSender sends a notification to a subscription.
type PushSender interface {
	SendPush(context.Context, PushSubscription, PushMessage) error
}

// App is a configured PWA integration.
type App struct {
	config        Config
	assets        *lazyassets.Registry
	workers       *lazyworkers.Registry
	manifest      []byte
	cacheManifest []byte
}

type options struct {
	appName string
	version string
	assets  *lazyassets.Registry
	workers *lazyworkers.Registry
}

// Option configures New.
type Option func(*options)

// WithAppName supplies the lazyapp.Config.Name fallback for manifest names.
func WithAppName(name string) Option {
	return func(options *options) {
		options.appName = name
	}
}

// WithVersion supplies the application build version fallback.
func WithVersion(version string) Option {
	return func(options *options) {
		options.version = version
	}
}

// WithAssets lets lazypwa add its browser client and resolve asset URLs.
func WithAssets(registry *lazyassets.Registry) Option {
	return func(options *options) {
		options.assets = registry
	}
}

// WithWorkers registers the PWA service worker in registry.
func WithWorkers(registry *lazyworkers.Registry) Option {
	return func(options *options) {
		options.workers = registry
	}
}

// IsEnabled reports whether config enables any PWA behavior.
func (config Config) IsEnabled() bool {
	return config.Installable ||
		config.Offline.Enabled ||
		config.Push.Enabled ||
		strings.TrimSpace(config.Manifest.Name) != "" ||
		strings.TrimSpace(config.Manifest.ShortName) != ""
}

// New creates a PWA integration and registers generated assets/workers.
func New(config Config, opts ...Option) (*App, error) {
	var settings options
	for _, option := range opts {
		option(&settings)
	}
	if strings.TrimSpace(settings.version) == "" {
		settings.version = lazybuildinfo.Version()
	}
	config = normalizeConfig(config, settings.appName, settings.version)
	app := &App{config: config, assets: settings.assets, workers: settings.workers}
	if !config.IsEnabled() {
		return app, nil
	}
	if settings.assets != nil {
		if err := settings.assets.Add(config.ClientAssetPath, []byte(app.clientScript()),
			lazyassets.ContentType("text/javascript; charset=utf-8"),
			lazyassets.AssetSource("lazypwa"),
		); err != nil {
			return nil, fmt.Errorf("register lazypwa client: %w", err)
		}
	}
	if settings.workers != nil {
		if err := settings.workers.AddScript(config.ServiceWorkerName, lazyworkers.ServiceWorker, config.ServiceWorkerPath, []byte(app.serviceWorkerScript()),
			lazyworkers.WithScope(config.Manifest.Scope),
			lazyworkers.WithScriptType(lazyworkers.ClassicScript),
			lazyworkers.WithDescription("GoLazy PWA service worker"),
			lazyworkers.WithPWA(),
		); err != nil {
			return nil, fmt.Errorf("register lazypwa service worker: %w", err)
		}
	}
	manifest, err := app.renderManifest()
	if err != nil {
		return nil, err
	}
	app.manifest = manifest
	cacheManifest, err := app.renderCacheManifest()
	if err != nil {
		return nil, err
	}
	app.cacheManifest = cacheManifest
	return app, nil
}

func normalizeConfig(config Config, appName string, version string) Config {
	if config.Version = strings.TrimSpace(config.Version); config.Version == "" || config.Version == "(devel)" {
		config.Version = firstNonEmpty(strings.TrimSpace(version), "devel")
	}
	config.ManifestPath = defaultPath(config.ManifestPath, defaultManifestPath)
	config.ServiceWorkerName = firstNonEmpty(strings.TrimSpace(config.ServiceWorkerName), defaultServiceWorkerName)
	config.ServiceWorkerPath = defaultPath(config.ServiceWorkerPath, defaultServiceWorkerPath)
	config.CacheManifestPath = defaultPath(config.CacheManifestPath, defaultCacheManifestPath)
	config.ClientAssetPath = defaultPath(config.ClientAssetPath, defaultClientAssetPath)
	config.Manifest.Name = firstNonEmpty(config.Manifest.Name, appName, "GoLazy App")
	config.Manifest.ShortName = firstNonEmpty(config.Manifest.ShortName, config.Manifest.Name)
	config.Manifest.StartURL = defaultPath(config.Manifest.StartURL, "/")
	config.Manifest.Scope = defaultPath(config.Manifest.Scope, "/")
	config.Manifest.Display = firstNonEmpty(config.Manifest.Display, "standalone")
	return config
}

func defaultPath(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = fallback
	}
	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	return path.Clean(value)
}

// Enabled reports whether the App has active PWA behavior.
func (app *App) Enabled() bool {
	return app != nil && app.config.IsEnabled()
}

// MiddlewareName returns the dispatcher-visible middleware name.
func (app *App) MiddlewareName() string {
	return "lazypwa.App"
}

// Handler serves generated PWA metadata and falls through to next for misses.
func (app *App) Handler(next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if app == nil || !app.Enabled() {
			next.ServeHTTP(w, r)
			return
		}
		switch r.URL.Path {
		case app.config.ManifestPath:
			serveJSONFile(w, r, app.manifest, "application/manifest+json")
		case app.config.CacheManifestPath:
			serveJSONFile(w, r, app.cacheManifest, "application/json; charset=utf-8")
		default:
			next.ServeHTTP(w, r)
		}
	})
}

func serveJSONFile(w http.ResponseWriter, r *http.Request, body []byte, contentType string) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=0, must-revalidate")
	if r.Method == http.MethodHead {
		return
	}
	_, _ = w.Write(body)
}

// ServeHTTP serves generated PWA metadata as a standalone handler.
func (app *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	app.Handler(http.NotFoundHandler()).ServeHTTP(w, r)
}

// Helpers returns lazyview-compatible PWA helpers.
func (app *App) Helpers() map[string]any {
	return map[string]any{
		"pwa": func() (lazyview.Fragment, error) {
			body, err := app.pwaTags()
			if err != nil {
				return lazyview.Fragment{}, err
			}
			return lazyview.Fragment{ContentType: "text/html; charset=utf-8", Body: body}, nil
		},
		"pwa_manifest": func() (lazyview.Fragment, error) {
			return lazyview.Fragment{
				ContentType: "text/html; charset=utf-8",
				Body:        `<link rel="manifest" href="` + html.EscapeString(app.config.ManifestPath) + `">`,
			}, nil
		},
		"pwa_client": func() (lazyview.Fragment, error) {
			src, err := app.clientPublicPath()
			if err != nil {
				return lazyview.Fragment{}, err
			}
			return lazyview.Fragment{
				ContentType: "text/html; charset=utf-8",
				Body:        `<script type="module" src="` + html.EscapeString(src) + `"></script>`,
			}, nil
		},
	}
}

func (app *App) pwaTags() (string, error) {
	client, err := app.clientPublicPath()
	if err != nil {
		return "", err
	}
	return `<link rel="manifest" href="` + html.EscapeString(app.config.ManifestPath) + `">` + "\n" +
		`<script type="module" src="` + html.EscapeString(client) + `"></script>`, nil
}

func (app *App) clientPublicPath() (string, error) {
	if app == nil {
		return "", fmt.Errorf("lazypwa: app is nil")
	}
	if app.assets == nil {
		return app.config.ClientAssetPath, nil
	}
	return app.assets.Path(app.config.ClientAssetPath)
}

type manifestDocument struct {
	Name            string   `json:"name"`
	ShortName       string   `json:"short_name,omitempty"`
	Description     string   `json:"description,omitempty"`
	StartURL        string   `json:"start_url"`
	Scope           string   `json:"scope,omitempty"`
	Display         string   `json:"display,omitempty"`
	ThemeColor      string   `json:"theme_color,omitempty"`
	BackgroundColor string   `json:"background_color,omitempty"`
	Orientation     string   `json:"orientation,omitempty"`
	Categories      []string `json:"categories,omitempty"`
	Icons           []Icon   `json:"icons,omitempty"`
}

func (app *App) renderManifest() ([]byte, error) {
	manifest := app.config.Manifest
	icons := make([]Icon, 0, len(manifest.Icons))
	for _, icon := range manifest.Icons {
		icon.Src = app.resolveAssetPath(icon.Src)
		icons = append(icons, icon)
	}
	doc := manifestDocument{
		Name:            manifest.Name,
		ShortName:       manifest.ShortName,
		Description:     manifest.Description,
		StartURL:        manifest.StartURL,
		Scope:           manifest.Scope,
		Display:         manifest.Display,
		ThemeColor:      manifest.ThemeColor,
		BackgroundColor: manifest.BackgroundColor,
		Orientation:     manifest.Orientation,
		Categories:      append([]string(nil), manifest.Categories...),
		Icons:           icons,
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("render web app manifest: %w", err)
	}
	return append(data, '\n'), nil
}

type cacheManifestDocument struct {
	Version  string   `json:"version"`
	URLs     []string `json:"urls"`
	Fallback string   `json:"fallback,omitempty"`
}

func (app *App) renderCacheManifest() ([]byte, error) {
	urls, err := app.offlineURLs()
	if err != nil {
		return nil, err
	}
	doc := cacheManifestDocument{
		Version:  app.config.Version,
		URLs:     urls,
		Fallback: app.config.Offline.FallbackURL,
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("render PWA cache manifest: %w", err)
	}
	return append(data, '\n'), nil
}

func (app *App) offlineURLs() ([]string, error) {
	if !app.config.Offline.Enabled {
		return nil, nil
	}
	seen := map[string]bool{}
	var urls []string
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			return
		}
		seen[value] = true
		urls = append(urls, value)
	}
	for _, url := range app.config.Offline.URLs {
		add(url)
	}
	for _, asset := range app.config.Offline.Assets {
		add(app.resolveAssetPath(asset))
	}
	if app.config.Offline.FallbackURL != "" {
		add(app.config.Offline.FallbackURL)
	}
	for _, source := range app.config.Offline.Sources {
		if source == nil {
			continue
		}
		sourceURLs, err := source.OfflineURLs()
		if err != nil {
			return nil, fmt.Errorf("load offline URLs: %w", err)
		}
		for _, url := range sourceURLs {
			add(url)
		}
	}
	if app.config.Offline.IncludeAssets && app.assets != nil {
		for _, asset := range app.assets.Manifest().Assets {
			if asset.Permanent != "" {
				add(asset.Permanent)
				continue
			}
			add(asset.Path)
		}
	}
	sort.Strings(urls)
	return urls, nil
}

func (app *App) resolveAssetPath(assetPath string) string {
	assetPath = strings.TrimSpace(assetPath)
	if assetPath == "" || app.assets == nil {
		return assetPath
	}
	resolved, err := app.assets.Path(assetPath)
	if err != nil {
		return assetPath
	}
	return resolved
}

func (app *App) serviceWorkerScript() string {
	cacheManifest := jsonString(app.config.CacheManifestPath)
	version := jsonString(app.config.Version)
	offline := "false"
	if app.config.Offline.Enabled {
		offline = "true"
	}
	return `"use strict";
const LAZYPWA_VERSION = ` + version + `;
const LAZYPWA_CACHE_MANIFEST = ` + cacheManifest + `;
const LAZYPWA_OFFLINE = ` + offline + `;
const LAZYPWA_CACHE = "lazypwa-" + LAZYPWA_VERSION;

self.addEventListener("install", (event) => {
  if (!LAZYPWA_OFFLINE) return;
  event.waitUntil(syncLazyPWACache());
});

self.addEventListener("activate", (event) => {
  event.waitUntil((async () => {
    const names = await caches.keys();
    await Promise.all(names.filter((name) => name.startsWith("lazypwa-") && name !== LAZYPWA_CACHE).map((name) => caches.delete(name)));
    await self.clients.claim();
  })());
});

self.addEventListener("message", (event) => {
  if (event.data && event.data.type === "lazypwa:activate-update") {
    self.skipWaiting();
  }
});

self.addEventListener("fetch", (event) => {
  if (!LAZYPWA_OFFLINE || event.request.method !== "GET") return;
  event.respondWith(fetch(event.request).catch(async () => {
    const cache = await caches.open(LAZYPWA_CACHE);
    const cached = await cache.match(event.request);
    if (cached) return cached;
    const manifest = await loadLazyPWAManifest();
    if (manifest.fallback) {
      const fallback = await cache.match(manifest.fallback);
      if (fallback) return fallback;
    }
    throw new Error("lazypwa: offline response unavailable");
  }));
});

async function syncLazyPWACache() {
  const manifest = await loadLazyPWAManifest();
  const cache = await caches.open(LAZYPWA_CACHE);
  if (Array.isArray(manifest.urls) && manifest.urls.length > 0) {
    await cache.addAll(manifest.urls);
  }
}

async function loadLazyPWAManifest() {
  const response = await fetch(LAZYPWA_CACHE_MANIFEST, { cache: "no-store" });
  return await response.json();
}
`
}

func (app *App) clientScript() string {
	workerPath := jsonString(app.config.ServiceWorkerPath)
	workerScope := jsonString(app.config.Manifest.Scope)
	workerType := jsonString(string(lazyworkers.ClassicScript))
	pushKey := jsonString(app.config.Push.ApplicationKey)
	return `"use strict";
(() => {
  const state = { deferredPrompt: null, registration: null };
  const dispatch = (name, detail = {}) => window.dispatchEvent(new CustomEvent("lazypwa:" + name, { detail }));

  window.addEventListener("beforeinstallprompt", (event) => {
    event.preventDefault();
    state.deferredPrompt = event;
    dispatch("installready");
  });
  window.addEventListener("offline", () => dispatch("offline"));
  window.addEventListener("online", () => dispatch("online"));

  window.lazypwa = {
    promptInstall: async () => {
      if (!state.deferredPrompt) return null;
      const prompt = state.deferredPrompt;
      state.deferredPrompt = null;
      prompt.prompt();
      return await prompt.userChoice;
    },
    checkForUpdate: async () => {
      if (!state.registration) return null;
      await state.registration.update();
      return state.registration;
    },
    activateUpdate: () => {
      if (state.registration && state.registration.waiting) {
        state.registration.waiting.postMessage({ type: "lazypwa:activate-update" });
      }
    },
    subscribePush: async () => {
      if (!state.registration || !state.registration.pushManager) return null;
      const key = ` + pushKey + `;
      const options = { userVisibleOnly: true };
      if (key) options.applicationServerKey = urlBase64ToUint8Array(key);
      const subscription = await state.registration.pushManager.subscribe(options);
      dispatch("pushsubscriptionchange", { subscription });
      return subscription;
    },
    unsubscribePush: async () => {
      if (!state.registration || !state.registration.pushManager) return false;
      const subscription = await state.registration.pushManager.getSubscription();
      if (!subscription) return false;
      const result = await subscription.unsubscribe();
      dispatch("pushsubscriptionchange", { subscription: null });
      return result;
    }
  };

  if ("serviceWorker" in navigator) {
    navigator.serviceWorker.register(` + workerPath + `, { scope: ` + workerScope + `, type: ` + workerType + ` }).then((registration) => {
      state.registration = registration;
      dispatch("ready", { registration });
      if (registration.waiting) dispatch("updateavailable", { registration });
      registration.addEventListener("updatefound", () => {
        const installing = registration.installing;
        if (!installing) return;
        installing.addEventListener("statechange", () => {
          if (installing.state === "installed" && navigator.serviceWorker.controller) {
            dispatch("updateavailable", { registration });
          }
        });
      });
    }).catch((error) => dispatch("error", { error }));
    navigator.serviceWorker.addEventListener("controllerchange", () => dispatch("controllerchange"));
  }

  function urlBase64ToUint8Array(value) {
    const padding = "=".repeat((4 - value.length % 4) % 4);
    const base64 = (value + padding).replace(/-/g, "+").replace(/_/g, "/");
    const raw = window.atob(base64);
    const output = new Uint8Array(raw.length);
    for (let i = 0; i < raw.length; i++) output[i] = raw.charCodeAt(i);
    return output;
  }
})();
`
}

// State is the lazydev-visible PWA state.
type State struct {
	Enabled           bool                `json:"enabled"`
	Installable       bool                `json:"installable"`
	Version           string              `json:"version,omitempty"`
	ManifestPath      string              `json:"manifest_path,omitempty"`
	ServiceWorkerName string              `json:"service_worker_name,omitempty"`
	ServiceWorkerPath string              `json:"service_worker_path,omitempty"`
	CacheManifestPath string              `json:"cache_manifest_path,omitempty"`
	ClientAssetPath   string              `json:"client_asset_path,omitempty"`
	Offline           OfflineState        `json:"offline"`
	Push              PushState           `json:"push"`
	ServiceWorker     *lazyworkers.Worker `json:"service_worker,omitempty"`
}

// OfflineState describes offline cache configuration.
type OfflineState struct {
	Enabled  bool   `json:"enabled"`
	URLs     int    `json:"urls"`
	Fallback string `json:"fallback,omitempty"`
}

// PushState describes push notification configuration.
type PushState struct {
	Enabled        bool `json:"enabled"`
	ApplicationKey bool `json:"application_key"`
	Store          bool `json:"store"`
	Sender         bool `json:"sender"`
}

// State returns the lazydev-visible PWA state.
func (app *App) State() State {
	if app == nil || !app.Enabled() {
		return State{}
	}
	urls, _ := app.offlineURLs()
	state := State{
		Enabled:           true,
		Installable:       app.config.Installable,
		Version:           app.config.Version,
		ManifestPath:      app.config.ManifestPath,
		ServiceWorkerName: app.config.ServiceWorkerName,
		ServiceWorkerPath: app.config.ServiceWorkerPath,
		CacheManifestPath: app.config.CacheManifestPath,
		ClientAssetPath:   app.config.ClientAssetPath,
		Offline: OfflineState{
			Enabled:  app.config.Offline.Enabled,
			URLs:     len(urls),
			Fallback: app.config.Offline.FallbackURL,
		},
		Push: PushState{
			Enabled:        app.config.Push.Enabled,
			ApplicationKey: strings.TrimSpace(app.config.Push.ApplicationKey) != "",
			Store:          app.config.Push.Store != nil,
			Sender:         app.config.Push.Sender != nil,
		},
	}
	if app.workers != nil {
		worker, ok := app.workers.Worker(app.config.ServiceWorkerName)
		if ok {
			state.ServiceWorker = &worker
		}
	}
	return state
}

func jsonString(value string) string {
	data, err := json.Marshal(value)
	if err != nil {
		return `""`
	}
	return string(data)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
