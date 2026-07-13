package lazyaddon

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// Hook is a typed lifecycle contract. Its ID and version form the stable
// cross-package identity; T is checked when callbacks are resolved.
type Hook[T any] struct {
	id      string
	version uint
	typeOf  reflect.Type
	catalog *Catalog
}

// ID returns the stable hook ID.
func (hook Hook[T]) ID() string { return hook.id }

// Version returns the hook contract version.
func (hook Hook[T]) Version() uint { return hook.version }

// CallbackOptions identifies and orders one hook callback.
type CallbackOptions struct {
	ID     string
	Before []string
	After  []string
}

type hookContract struct {
	id      string
	version uint
	typeOf  reflect.Type
}

type registeredCallback struct {
	addonID string
	owner   *registrationToken
	id      string
	hookID  string
	version uint
	typeOf  reflect.Type
	before  []string
	after   []string
	run     func(any) error
}

// DefineHook defines a hook contract in the default process catalog.
func DefineHook[T any](id string, version uint) Hook[T] {
	return DefineHookIn[T](defaultCatalog, id, version)
}

// DefineHookIn defines a hook contract in catalog.
func DefineHookIn[T any](catalog *Catalog, id string, version uint) Hook[T] {
	if catalog == nil {
		panic("lazyaddon: catalog is nil")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		panic("lazyaddon: hook ID is required")
	}
	if version == 0 {
		panic("lazyaddon: hook version is required")
	}
	contract := hookContract{id: id, version: version, typeOf: eventType[T]()}
	catalog.mu.Lock()
	if previous, exists := catalog.hooks[id]; exists {
		if previous.version != contract.version || previous.typeOf != contract.typeOf {
			catalog.mu.Unlock()
			panic(fmt.Sprintf("lazyaddon: hook %q is already defined with a different contract", id))
		}
	} else {
		catalog.hooks[id] = contract
	}
	catalog.mu.Unlock()
	return Hook[T]{id: id, version: version, typeOf: contract.typeOf, catalog: catalog}
}

// On registers callback for hook in the hook's catalog. Registration proves
// which add-on owns the callback; callbacks run only for application scopes
// that select that add-on.
func On[T any](registration Registration, hook Hook[T], options CallbackOptions, callback func(*T) error) error {
	catalog := hook.catalog
	return OnIn(catalog, registration, hook, options, callback)
}

// MustOn is On with panic-on-error semantics for use from init functions.
func MustOn[T any](registration Registration, hook Hook[T], options CallbackOptions, callback func(*T) error) {
	if err := On(registration, hook, options, callback); err != nil {
		panic(err)
	}
}

// OnIn registers a callback in an explicit catalog.
func OnIn[T any](catalog *Catalog, registration Registration, hook Hook[T], options CallbackOptions, callback func(*T) error) error {
	if catalog == nil {
		return fmt.Errorf("lazyaddon: catalog is nil")
	}
	if hook.id == "" || hook.version == 0 {
		return fmt.Errorf("lazyaddon: callback hook contract is invalid")
	}
	if hook.catalog != catalog {
		return fmt.Errorf("lazyaddon: hook %q belongs to a different catalog", hook.id)
	}
	if callback == nil {
		return fmt.Errorf("lazyaddon: callback for hook %q is nil", hook.id)
	}
	callbackID := strings.TrimSpace(options.ID)
	if callbackID == "" {
		return fmt.Errorf("lazyaddon: callback ID is required for hook %q", hook.id)
	}

	catalog.mu.Lock()
	defer catalog.mu.Unlock()
	if err := catalog.validateRegistrationLocked(registration); err != nil {
		return fmt.Errorf("lazyaddon: callback owner: %w", err)
	}
	contract, exists := catalog.hooks[hook.id]
	if !exists || contract.version != hook.version || contract.typeOf != eventType[T]() {
		return fmt.Errorf("lazyaddon: callback hook %q contract mismatch", hook.id)
	}
	addonID := registration.id
	if !strings.Contains(callbackID, "/") {
		callbackID = addonID + "/" + callbackID
	}
	registered := registeredCallback{
		addonID: addonID,
		owner:   registration.token,
		id:      callbackID,
		hookID:  hook.id,
		version: hook.version,
		typeOf:  eventType[T](),
		before:  normalizeIDs(options.Before),
		after:   normalizeIDs(options.After),
		run: func(event any) error {
			typed, ok := event.(*T)
			if !ok {
				return fmt.Errorf("lazyaddon: hook %q received %T, want *%s", hook.id, event, eventType[T]())
			}
			return callback(typed)
		},
	}

	for _, existing := range catalog.callbacks[hook.id] {
		if existing.id == registered.id {
			return fmt.Errorf("lazyaddon: callback %q is already registered for hook %q", registered.id, hook.id)
		}
	}
	catalog.callbacks[hook.id] = append(catalog.callbacks[hook.id], registered)
	return nil
}

// Run executes active callbacks for hook in deterministic Before/After order.
func Run[T any](scope *Scope, hook Hook[T], event *T) error {
	if scope == nil {
		return fmt.Errorf("lazyaddon: scope is nil")
	}
	if event == nil {
		return fmt.Errorf("lazyaddon: event for hook %q is nil", hook.id)
	}
	contract, exists := scope.hooks[hook.id]
	if !exists {
		return fmt.Errorf("lazyaddon: hook %q is not defined", hook.id)
	}
	if contract.version != hook.version || contract.typeOf != eventType[T]() {
		return fmt.Errorf("lazyaddon: hook %q contract mismatch", hook.id)
	}

	callbacks := make([]registeredCallback, 0, len(scope.callbacks[hook.id]))
	for _, callback := range scope.callbacks[hook.id] {
		if scope.registrations[callback.addonID] != callback.owner {
			continue
		}
		if callback.version != contract.version || callback.typeOf != contract.typeOf {
			return fmt.Errorf("lazyaddon: callback %q does not match hook %q", callback.id, hook.id)
		}
		callbacks = append(callbacks, callback)
	}
	ordered, err := orderCallbacks(callbacks, scope.definitions)
	if err != nil {
		return fmt.Errorf("lazyaddon: hook %q: %w", hook.id, err)
	}
	for _, callback := range ordered {
		if err := callback.run(event); err != nil {
			return fmt.Errorf("lazyaddon: hook %q callback %q: %w", hook.id, callback.id, err)
		}
	}
	return nil
}

func orderCallbacks(callbacks []registeredCallback, definitions map[string]Definition) ([]registeredCallback, error) {
	byID := make(map[string]registeredCallback, len(callbacks))
	for _, callback := range callbacks {
		if _, exists := byID[callback.id]; exists {
			return nil, fmt.Errorf("callback %q is duplicated", callback.id)
		}
		byID[callback.id] = callback
	}
	edges := make(map[string]map[string]bool, len(callbacks))
	indegree := make(map[string]int, len(callbacks))
	for id := range byID {
		edges[id] = map[string]bool{}
		indegree[id] = 0
	}
	addEdge := func(from, to string) {
		if _, exists := byID[from]; !exists {
			return
		}
		if _, exists := byID[to]; !exists || edges[from][to] {
			return
		}
		edges[from][to] = true
		indegree[to]++
	}
	for _, callback := range callbacks {
		for _, before := range callback.before {
			addEdge(callback.id, resolveOrderingTarget(callback.addonID, before, byID))
		}
		for _, after := range callback.after {
			addEdge(resolveOrderingTarget(callback.addonID, after, byID), callback.id)
		}
	}
	// Runtime callbacks inherit the resolved add-on graph's dependency order.
	// A dependent add-on may refine that order with Before/After, but it cannot
	// force its callback ahead of a required add-on without creating a cycle.
	for _, dependent := range callbacks {
		for _, dependency := range callbacks {
			if dependent.addonID == dependency.addonID || !addonDependsOn(definitions, dependent.addonID, dependency.addonID, nil) {
				continue
			}
			addEdge(dependency.id, dependent.id)
		}
	}

	ready := make([]string, 0, len(callbacks))
	for id, degree := range indegree {
		if degree == 0 {
			ready = append(ready, id)
		}
	}
	sort.Strings(ready)
	ordered := make([]registeredCallback, 0, len(callbacks))
	for len(ready) > 0 {
		id := ready[0]
		ready = ready[1:]
		ordered = append(ordered, byID[id])
		for _, target := range sortedKeys(edges[id]) {
			indegree[target]--
			if indegree[target] == 0 {
				ready = append(ready, target)
				sort.Strings(ready)
			}
		}
	}
	if len(ordered) != len(callbacks) {
		var cyclic []string
		for id, degree := range indegree {
			if degree > 0 {
				cyclic = append(cyclic, id)
			}
		}
		sort.Strings(cyclic)
		return nil, fmt.Errorf("callback ordering cycle involving %s", strings.Join(cyclic, ", "))
	}
	return ordered, nil
}

func addonDependsOn(definitions map[string]Definition, addonID, dependencyID string, visiting map[string]bool) bool {
	if addonID == dependencyID {
		return false
	}
	if visiting == nil {
		visiting = map[string]bool{}
	}
	if visiting[addonID] {
		return false
	}
	visiting[addonID] = true
	defer delete(visiting, addonID)
	for _, requirement := range definitions[addonID].Requires {
		required, _ := splitRequirement(requirement)
		if required == dependencyID || addonDependsOn(definitions, required, dependencyID, visiting) {
			return true
		}
	}
	return false
}

func resolveOrderingTarget(addonID, target string, callbacks map[string]registeredCallback) string {
	target = strings.TrimSpace(target)
	if _, exists := callbacks[target]; exists {
		return target
	}
	local := addonID + "/" + target
	if _, exists := callbacks[local]; exists {
		return local
	}
	return target
}

func eventType[T any]() reflect.Type {
	return reflect.TypeOf((*T)(nil)).Elem()
}

func cloneHooks(source map[string]hookContract) map[string]hookContract {
	out := make(map[string]hookContract, len(source))
	for id, hook := range source {
		out[id] = hook
	}
	return out
}

func cloneCallbacks(source map[string][]registeredCallback) map[string][]registeredCallback {
	out := make(map[string][]registeredCallback, len(source))
	for hook, callbacks := range source {
		cloned := make([]registeredCallback, len(callbacks))
		copy(cloned, callbacks)
		for index := range cloned {
			cloned[index].before = append([]string(nil), cloned[index].before...)
			cloned[index].after = append([]string(nil), cloned[index].after...)
		}
		out[hook] = cloned
	}
	return out
}
