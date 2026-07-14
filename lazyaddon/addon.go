package lazyaddon

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Definition describes one add-on exposed by an add-on package.
type Definition struct {
	ID          string
	Version     string
	Description string
	// Requires accepts id@version. A bare ID is normalized to this
	// definition's Version, which is convenient for add-ons in one package.
	Requires  []string
	Optional  []string
	Conflicts []string
}

// Use selects one add-on for an application. Version is optional for manual
// selections; generated installer wiring always sets it from addons.toml.
type Use struct {
	ID      string
	Version string
	Config  map[string]string
}

// Selection is the set of add-ons explicitly selected by an application.
// Required add-ons are resolved automatically.
type Selection struct {
	Addons []Use
}

// Select constructs a Selection from add-on IDs.
func Select(ids ...string) Selection {
	selection := Selection{Addons: make([]Use, 0, len(ids))}
	for _, id := range ids {
		if id = strings.TrimSpace(id); id != "" {
			selection.Addons = append(selection.Addons, Use{ID: id})
		}
	}
	return selection
}

// Package registers all add-ons declared by one lazyaddon.toml manifest.
type Package struct {
	Manifest Manifest
}

// Registration is an opaque proof that an add-on definition was registered
// by a catalog. Add-on packages should keep their Registration private and use
// it when defining capabilities and registering callbacks.
//
// A Registration may be copied, but its zero value is invalid.
type Registration struct {
	catalog *Catalog
	id      string
	token   *registrationToken
}

// ID returns the registered add-on ID.
func (registration Registration) ID() string { return registration.id }

// registrationToken has non-zero size so distinct catalog-issued tokens always
// have distinct addresses.
type registrationToken struct {
	marker byte
}

// Catalog stores add-on definitions, hook contracts, and callbacks.
// Registrations are process-wide by default, while resolved scopes are
// immutable and application-local.
type Catalog struct {
	mu            sync.RWMutex
	definitions   map[string]Definition
	registrations map[string]*registrationToken
	hooks         map[string]hookContract
	capabilities  map[string]capabilityContract
	callbacks     map[string][]registeredCallback
}

// NewCatalog creates an empty catalog. Most add-on packages use the default
// process catalog through MustRegister and On; explicit catalogs are useful in
// tests and applications that need complete registry isolation.
func NewCatalog() *Catalog {
	return &Catalog{
		definitions:   map[string]Definition{},
		registrations: map[string]*registrationToken{},
		hooks:         map[string]hookContract{},
		capabilities:  map[string]capabilityContract{},
		callbacks:     map[string][]registeredCallback{},
	}
}

var defaultCatalog = NewCatalog()

// MustRegister registers every definition in pkg's manifest in the default
// process catalog, returns their opaque registrations, and panics on invalid
// or duplicate definitions.
func MustRegister(pkg Package) []Registration {
	registrations, err := defaultCatalog.RegisterPackage(pkg)
	if err != nil {
		panic(err)
	}
	return registrations
}

// MustRegisterDefinition registers one definition in the default process
// catalog, returns its opaque registration, and panics on failure.
func MustRegisterDefinition(definition Definition) Registration {
	registration, err := defaultCatalog.Register(definition)
	if err != nil {
		panic(err)
	}
	return registration
}

// RegisterPackage registers all definitions in pkg and returns their opaque
// registrations in manifest order.
func (catalog *Catalog) RegisterPackage(pkg Package) ([]Registration, error) {
	if catalog == nil {
		return nil, fmt.Errorf("lazyaddon: catalog is nil")
	}
	if err := pkg.Manifest.Validate(); err != nil {
		return nil, err
	}
	registrations := make([]Registration, 0, len(pkg.Manifest.Addons))
	for _, definition := range pkg.Manifest.Addons {
		if definition.Version == "" {
			definition.Version = pkg.Manifest.Package.Version
		}
		registration, err := catalog.Register(definition)
		if err != nil {
			return nil, err
		}
		registrations = append(registrations, registration)
	}
	return registrations, nil
}

// Register registers one add-on definition and returns its opaque
// registration.
func (catalog *Catalog) Register(definition Definition) (Registration, error) {
	if catalog == nil {
		return Registration{}, fmt.Errorf("lazyaddon: catalog is nil")
	}
	definition = normalizeDefinition(definition)
	if err := validateDefinition(definition); err != nil {
		return Registration{}, err
	}

	catalog.mu.Lock()
	defer catalog.mu.Unlock()
	if previous, exists := catalog.definitions[definition.ID]; exists {
		return Registration{}, fmt.Errorf("lazyaddon: add-on %q is already registered at version %q", definition.ID, previous.Version)
	}
	token := &registrationToken{}
	catalog.definitions[definition.ID] = definition
	catalog.registrations[definition.ID] = token
	return Registration{catalog: catalog, id: definition.ID, token: token}, nil
}

// Scope is an immutable resolved add-on graph for one application.
type Scope struct {
	catalog       *Catalog
	definitions   map[string]Definition
	registrations map[string]*registrationToken
	uses          map[string]Use
	order         []string
	hooks         map[string]hookContract
	callbacks     map[string][]registeredCallback
	capabilities  map[string]capabilityContract
	capabilityMu  sync.RWMutex
	values        map[string]any
}

// Resolve resolves selection against the default process catalog.
func Resolve(selection Selection) (*Scope, error) {
	return defaultCatalog.Resolve(selection)
}

// Resolve validates and resolves selection, including required dependencies.
func (catalog *Catalog) Resolve(selection Selection) (*Scope, error) {
	if catalog == nil {
		return nil, fmt.Errorf("lazyaddon: catalog is nil")
	}
	catalog.mu.RLock()
	definitions := cloneDefinitions(catalog.definitions)
	registrations := cloneRegistrations(catalog.registrations)
	hooks := cloneHooks(catalog.hooks)
	capabilities := cloneCapabilities(catalog.capabilities)
	callbacks := cloneCallbacks(catalog.callbacks)
	catalog.mu.RUnlock()

	uses := map[string]Use{}
	requested := make([]string, 0, len(selection.Addons))
	for _, use := range selection.Addons {
		use.ID = strings.TrimSpace(use.ID)
		use.Version = strings.TrimSpace(use.Version)
		if use.ID == "" {
			return nil, fmt.Errorf("lazyaddon: selected add-on ID is required")
		}
		if _, exists := uses[use.ID]; exists {
			return nil, fmt.Errorf("lazyaddon: add-on %q is selected more than once", use.ID)
		}
		use.Config = cloneStringMap(use.Config)
		uses[use.ID] = use
		requested = append(requested, use.ID)
	}

	state := map[string]uint8{}
	var order []string
	var visit func(string, []string) error
	visit = func(id string, trail []string) error {
		definition, exists := definitions[id]
		if !exists {
			return fmt.Errorf("lazyaddon: add-on %q is not registered", id)
		}
		if expected := uses[id].Version; expected != "" && definition.Version != expected {
			return fmt.Errorf("lazyaddon: add-on %q registered version %q does not match selected version %q", id, definition.Version, expected)
		}
		switch state[id] {
		case 1:
			cycle := append(append([]string(nil), trail...), id)
			return fmt.Errorf("lazyaddon: dependency cycle: %s", strings.Join(cycle, " -> "))
		case 2:
			return nil
		}
		state[id] = 1
		for _, requirement := range definition.Requires {
			required, version := splitRequirement(requirement)
			if use, exists := uses[required]; !exists {
				uses[required] = Use{ID: required, Version: version}
			} else if version != "" && use.Version != "" && use.Version != version {
				return fmt.Errorf("lazyaddon: add-on %q requires %q at version %q, but version %q is selected", id, required, version, use.Version)
			} else if version != "" && use.Version == "" {
				use.Version = version
				uses[required] = use
			}
			if err := visit(required, append(trail, id)); err != nil {
				return err
			}
		}
		state[id] = 2
		order = append(order, id)
		return nil
	}
	for _, id := range requested {
		if err := visit(id, nil); err != nil {
			return nil, err
		}
	}

	for id := range uses {
		definition := definitions[id]
		for _, conflict := range definition.Conflicts {
			if _, selected := uses[conflict]; selected {
				return nil, fmt.Errorf("lazyaddon: add-on %q conflicts with %q", id, conflict)
			}
		}
	}

	selectedDefinitions := make(map[string]Definition, len(uses))
	selectedRegistrations := make(map[string]*registrationToken, len(uses))
	for id := range uses {
		selectedDefinitions[id] = definitions[id]
		selectedRegistrations[id] = registrations[id]
	}
	return &Scope{
		catalog:       catalog,
		definitions:   selectedDefinitions,
		registrations: selectedRegistrations,
		uses:          uses,
		order:         order,
		hooks:         hooks,
		callbacks:     callbacks,
		capabilities:  capabilities,
		values:        map[string]any{},
	}, nil
}

// Addons returns selected and required add-on IDs in dependency-first order.
func (scope *Scope) Addons() []string {
	if scope == nil {
		return nil
	}
	return append([]string(nil), scope.order...)
}

// Has reports whether id is active in this application scope.
func (scope *Scope) Has(id string) bool {
	if scope == nil {
		return false
	}
	_, ok := scope.uses[strings.TrimSpace(id)]
	return ok
}

// HasCallbacks reports whether at least one active add-on registered a
// callback for hook. It lets lifecycle owners avoid allocating optional
// subsystem state when no selected add-on contributes to that phase.
func (scope *Scope) HasCallbacks(hookID string) bool {
	if scope == nil {
		return false
	}
	for _, callback := range scope.callbacks[strings.TrimSpace(hookID)] {
		if scope.registrations[callback.addonID] == callback.owner {
			return true
		}
	}
	return false
}

// Config returns a copy of the selected add-on's non-secret configuration.
func (scope *Scope) Config(id string) map[string]string {
	if scope == nil {
		return nil
	}
	return cloneStringMap(scope.uses[strings.TrimSpace(id)].Config)
}

func normalizeDefinition(definition Definition) Definition {
	definition.ID = strings.TrimSpace(definition.ID)
	definition.Version = strings.TrimSpace(definition.Version)
	definition.Description = strings.TrimSpace(definition.Description)
	definition.Requires = normalizeRequirements(definition.Requires, definition.Version)
	definition.Optional = normalizeIDs(definition.Optional)
	definition.Conflicts = normalizeIDs(definition.Conflicts)
	return definition
}

func normalizeRequirements(requirements []string, defaultVersion string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(requirements))
	for _, requirement := range requirements {
		id, version := splitRequirement(requirement)
		if version == "" {
			version = strings.TrimSpace(defaultVersion)
		}
		if id == "" {
			continue
		}
		normalized := id
		if version != "" {
			normalized += "@" + version
		}
		if !seen[normalized] {
			seen[normalized] = true
			out = append(out, normalized)
		}
	}
	return out
}

func normalizeIDs(ids []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if id = strings.TrimSpace(id); id != "" && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

func validateDefinition(definition Definition) error {
	if definition.ID == "" {
		return fmt.Errorf("lazyaddon: add-on ID is required")
	}
	if !validAddonID(definition.ID) {
		return fmt.Errorf("lazyaddon: add-on ID %q contains an invalid character or path segment", definition.ID)
	}
	if definition.Version == "" {
		return fmt.Errorf("lazyaddon: add-on %q version is required", definition.ID)
	}
	for _, conflict := range definition.Conflicts {
		if conflict == definition.ID {
			return fmt.Errorf("lazyaddon: add-on %q cannot conflict with itself", definition.ID)
		}
	}
	for _, requirement := range definition.Requires {
		required, version := splitRequirement(requirement)
		if !validAddonID(required) || strings.Contains(requirement, "@") && version == "" {
			return fmt.Errorf("lazyaddon: add-on %q has invalid requirement %q", definition.ID, requirement)
		}
		if required == definition.ID {
			return fmt.Errorf("lazyaddon: add-on %q cannot require itself", definition.ID)
		}
	}
	for _, related := range append(append([]string(nil), definition.Optional...), definition.Conflicts...) {
		if !validAddonID(related) {
			return fmt.Errorf("lazyaddon: add-on %q references invalid add-on ID %q", definition.ID, related)
		}
	}
	return nil
}

func splitRequirement(requirement string) (id, version string) {
	requirement = strings.TrimSpace(requirement)
	if at := strings.LastIndexByte(requirement, '@'); at > 0 {
		return strings.TrimSpace(requirement[:at]), strings.TrimSpace(requirement[at+1:])
	}
	return requirement, ""
}

func validAddonID(id string) bool {
	if strings.HasPrefix(id, "/") || strings.HasSuffix(id, "/") || strings.Contains(id, "//") {
		return false
	}
	for _, part := range strings.Split(id, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
		for _, character := range part {
			if character >= 'a' && character <= 'z' ||
				character >= '0' && character <= '9' ||
				strings.ContainsRune("._-", character) {
				continue
			}
			return false
		}
	}
	return true
}

func cloneDefinitions(source map[string]Definition) map[string]Definition {
	out := make(map[string]Definition, len(source))
	for id, definition := range source {
		definition.Requires = append([]string(nil), definition.Requires...)
		definition.Optional = append([]string(nil), definition.Optional...)
		definition.Conflicts = append([]string(nil), definition.Conflicts...)
		out[id] = definition
	}
	return out
}

func cloneRegistrations(source map[string]*registrationToken) map[string]*registrationToken {
	out := make(map[string]*registrationToken, len(source))
	for id, registration := range source {
		out[id] = registration
	}
	return out
}

func (catalog *Catalog) validateRegistrationLocked(registration Registration) error {
	if registration.catalog == nil || registration.id == "" || registration.token == nil {
		return fmt.Errorf("lazyaddon: registration is invalid")
	}
	if registration.catalog != catalog {
		return fmt.Errorf("lazyaddon: registration for add-on %q belongs to a different catalog", registration.id)
	}
	token, exists := catalog.registrations[registration.id]
	if !exists || token != registration.token {
		return fmt.Errorf("lazyaddon: registration does not own add-on %q", registration.id)
	}
	return nil
}

func cloneStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	out := make(map[string]string, len(source))
	for key, value := range source {
		out[key] = value
	}
	return out
}

func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
