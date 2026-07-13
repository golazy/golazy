package lazyaddon

import (
	"fmt"
	"reflect"
	"strings"
)

// Capability is a typed, versioned value exchanged by add-ons in one
// application scope.
type Capability[T any] struct {
	id         string
	version    uint
	typeOf     reflect.Type
	catalog    *Catalog
	ownerID    string
	ownerToken *registrationToken
}

// ID returns the stable capability ID.
func (capability Capability[T]) ID() string { return capability.id }

// Version returns the capability contract version.
func (capability Capability[T]) Version() uint { return capability.version }

type capabilityContract struct {
	id         string
	version    uint
	typeOf     reflect.Type
	ownerID    string
	ownerToken *registrationToken
}

// DefineCapability defines a capability owned by registration in its catalog.
func DefineCapability[T any](registration Registration, id string, version uint) Capability[T] {
	return DefineCapabilityIn[T](registration.catalog, registration, id, version)
}

// DefineCapabilityIn defines a capability owned by registration in catalog.
func DefineCapabilityIn[T any](catalog *Catalog, registration Registration, id string, version uint) Capability[T] {
	if catalog == nil {
		panic("lazyaddon: catalog is nil")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		panic("lazyaddon: capability ID is required")
	}
	if version == 0 {
		panic("lazyaddon: capability version is required")
	}
	catalog.mu.Lock()
	if err := catalog.validateRegistrationLocked(registration); err != nil {
		catalog.mu.Unlock()
		panic(fmt.Sprintf("lazyaddon: capability owner: %v", err))
	}
	contract := capabilityContract{
		id:         id,
		version:    version,
		typeOf:     eventType[T](),
		ownerID:    registration.id,
		ownerToken: registration.token,
	}
	if previous, exists := catalog.capabilities[id]; exists {
		if previous.version != contract.version || previous.typeOf != contract.typeOf ||
			previous.ownerID != contract.ownerID || previous.ownerToken != contract.ownerToken {
			catalog.mu.Unlock()
			panic(fmt.Sprintf("lazyaddon: capability %q is already defined with a different contract", id))
		}
	} else {
		catalog.capabilities[id] = contract
	}
	catalog.mu.Unlock()
	return Capability[T]{
		id:         id,
		version:    version,
		typeOf:     contract.typeOf,
		catalog:    catalog,
		ownerID:    registration.id,
		ownerToken: registration.token,
	}
}

// Provide stores value in scope. A capability has exactly one provider per
// application scope, and only its defining add-on registration may provide it.
func Provide[T any](scope *Scope, registration Registration, capability Capability[T], value T) error {
	if scope == nil {
		return fmt.Errorf("lazyaddon: scope is nil")
	}
	if registration.catalog == nil || registration.id == "" || registration.token == nil {
		return fmt.Errorf("lazyaddon: capability provider registration is invalid")
	}
	if registration.catalog != scope.catalog {
		return fmt.Errorf("lazyaddon: capability provider registration for add-on %q belongs to a different catalog", registration.id)
	}
	if scope.registrations[registration.id] != registration.token {
		return fmt.Errorf("lazyaddon: capability provider registration does not own selected add-on %q", registration.id)
	}
	if capability.catalog != registration.catalog || capability.ownerID != registration.id || capability.ownerToken != registration.token {
		return fmt.Errorf("lazyaddon: add-on %q does not own capability %q", registration.id, capability.id)
	}
	contract, exists := scope.capabilities[capability.id]
	if !exists {
		return fmt.Errorf("lazyaddon: capability %q is not defined", capability.id)
	}
	if contract.version != capability.version || contract.typeOf != eventType[T]() ||
		contract.ownerID != capability.ownerID || contract.ownerToken != capability.ownerToken {
		return fmt.Errorf("lazyaddon: capability %q contract mismatch", capability.id)
	}
	scope.capabilityMu.Lock()
	defer scope.capabilityMu.Unlock()
	if _, exists := scope.values[capability.id]; exists {
		return fmt.Errorf("lazyaddon: capability %q already has a provider", capability.id)
	}
	scope.values[capability.id] = value
	return nil
}

// Require returns a capability value from scope.
func Require[T any](scope *Scope, capability Capability[T]) (T, error) {
	var zero T
	if scope == nil {
		return zero, fmt.Errorf("lazyaddon: scope is nil")
	}
	contract, exists := scope.capabilities[capability.id]
	if !exists || contract.version != capability.version || contract.typeOf != eventType[T]() ||
		contract.ownerID != capability.ownerID || contract.ownerToken != capability.ownerToken {
		return zero, fmt.Errorf("lazyaddon: capability %q contract mismatch", capability.id)
	}
	scope.capabilityMu.RLock()
	value, exists := scope.values[capability.id]
	scope.capabilityMu.RUnlock()
	if !exists {
		return zero, fmt.Errorf("lazyaddon: capability %q is not available", capability.id)
	}
	typed, ok := value.(T)
	if !ok {
		return zero, fmt.Errorf("lazyaddon: capability %q contains %T, want %s", capability.id, value, eventType[T]())
	}
	return typed, nil
}

func cloneCapabilities(source map[string]capabilityContract) map[string]capabilityContract {
	out := make(map[string]capabilityContract, len(source))
	for id, capability := range source {
		out[id] = capability
	}
	return out
}
