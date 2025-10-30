package typeregistry

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"sync"
)

// --- Errors --------------------------------------------------------

var (
	ErrTypeNotValid      = errors.New("typeregistry: invalid type")
	ErrTypeAlreadyExists = errors.New("typeregistry: type already registered")
	ErrTypeNotRegistered = errors.New("typeregistry: type not registered")
	ErrMarshal           = errors.New("typeregistry: marshal error")
	ErrUnmarshal         = errors.New("typeregistry: unmarshal error")

	nameRegex = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)
)

// --- Registry ------------------------------------------------------

// TypeInfo holds metadata about a registered type
type TypeInfo struct {
	Type     reflect.Type
	Metadata map[string]interface{}
	Aliases  []string
	Validate func(any) error // Optional validation function
}

// TypedData represents a value with type information, following CloudEvents pattern
// This structure enables type-safe JSON marshaling/unmarshaling with embedded type metadata
type TypedData struct {
	Type string          `json:"type"`           // Type identifier (e.g., "app.User")
	Data json.RawMessage `json:"data"`           // The actual data payload
}

// NewTypedData creates a TypedData from a type name and raw JSON data
func NewTypedData(typeName string, data json.RawMessage) *TypedData {
	return &TypedData{
		Type: typeName,
		Data: data,
	}
}

// MarshalValue creates TypedData by marshaling the data field
func (td *TypedData) MarshalValue(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}
	td.Data = data
	return nil
}

// UnmarshalValue unmarshals the data field into the provided value
func (td *TypedData) UnmarshalValue(v any) error {
	if err := json.Unmarshal(td.Data, v); err != nil {
		return fmt.Errorf("failed to unmarshal value: %w", err)
	}
	return nil
}

type Registry struct {
	mu        sync.RWMutex
	types     map[string]*TypeInfo           // name -> TypeInfo
	rtypes    map[reflect.Type]string        // reverse lookup: type -> primary name
	aliases   map[string]string              // alias -> primary name
	jsonCache sync.Map                       // Cache for JSON schemas
}

func New() *Registry {
	return &Registry{
		types:   make(map[string]*TypeInfo),
		rtypes:  make(map[reflect.Type]string),
		aliases: make(map[string]string),
	}
}

// --- Registration --------------------------------------------------

// inferTypeName generates a name from the type's package and struct name
func inferTypeName(rt reflect.Type) string {
	// Get the element type (dereference pointer)
	elemType := rt.Elem()

	// Get package name and type name
	pkgPath := elemType.PkgPath()
	typeName := elemType.Name()

	// If no package (main package), just use the type name
	if pkgPath == "" {
		return typeName
	}

	// Extract just the package name from the full path
	// For example: "github.com/user/project/pkg" -> "pkg"
	parts := strings.Split(pkgPath, "/")
	pkgName := parts[len(parts)-1]

	return pkgName + "." + typeName
}

// normalizeType returns the element type for pointers, or the type itself otherwise
func normalizeType(rt reflect.Type) reflect.Type {
	if rt.Kind() == reflect.Ptr {
		return rt.Elem()
	}
	return rt
}

// internal non-generic registration logic
func (r *Registry) register(name string, rt reflect.Type) error {
	return r.registerWithOptions(name, rt, nil, nil)
}

// registerWithOptions registers a type with optional metadata and validation
func (r *Registry) registerWithOptions(name string, rt reflect.Type, metadata map[string]interface{}, validate func(any) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// If name is empty, infer it from the type
	if name == "" {
		name = inferTypeName(rt)
	}

	if !nameRegex.MatchString(name) {
		return fmt.Errorf("%w: invalid name %q", ErrTypeNotValid, name)
	}

	if rt.Kind() != reflect.Ptr || rt.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("%w: %s must be a pointer to a struct", ErrTypeNotValid, rt)
	}

	if _, exists := r.types[name]; exists {
		return fmt.Errorf("%w: %s", ErrTypeAlreadyExists, name)
	}

	info := &TypeInfo{
		Type:     rt,
		Metadata: metadata,
		Validate: validate,
		Aliases:  []string{},
	}

	r.types[name] = info
	// Store only the normalized (element) type to save memory
	r.rtypes[normalizeType(rt)] = name

	return nil
}

// Register registers a Go type under a given name.
func Register[T any](r *Registry, name string) error {
	var zero T
	rt := reflect.TypeOf(&zero)
	return r.register(name, rt)
}

// MustRegister panics if registration fails.
func MustRegister[T any](r *Registry, name string) {
	if err := Register[T](r, name); err != nil {
		panic(err)
	}
}

// RegisterWithMetadata registers a type with metadata
func RegisterWithMetadata[T any](r *Registry, name string, metadata map[string]interface{}) error {
	var zero T
	rt := reflect.TypeOf(&zero)
	return r.registerWithOptions(name, rt, metadata, nil)
}

// RegisterWithValidation registers a type with a validation function
func RegisterWithValidation[T any](r *Registry, name string, validate func(any) error) error {
	var zero T
	rt := reflect.TypeOf(&zero)
	return r.registerWithOptions(name, rt, nil, validate)
}

// AddAlias adds an alias for an existing type
func (r *Registry) AddAlias(alias, primaryName string) error {
	if r == nil {
		return fmt.Errorf("typeregistry: nil registry")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if !nameRegex.MatchString(alias) {
		return fmt.Errorf("%w: invalid alias %q", ErrTypeNotValid, alias)
	}

	info, exists := r.types[primaryName]
	if !exists {
		return fmt.Errorf("%w: primary type %s", ErrTypeNotRegistered, primaryName)
	}

	if _, exists := r.aliases[alias]; exists {
		return fmt.Errorf("%w: alias %s", ErrTypeAlreadyExists, alias)
	}

	if _, exists := r.types[alias]; exists {
		return fmt.Errorf("%w: alias conflicts with existing type %s", ErrTypeAlreadyExists, alias)
	}

	r.aliases[alias] = primaryName
	info.Aliases = append(info.Aliases, alias)

	return nil
}

// TypeEntry represents a type and its name for batch registration
type TypeEntry struct {
	Name     string
	Type     reflect.Type
	Metadata map[string]interface{}
	Validate func(any) error
}

// RegisterBatch registers multiple types at once with a single lock acquisition
func (r *Registry) RegisterBatch(entries []TypeEntry) error {
	if r == nil {
		return fmt.Errorf("typeregistry: nil registry")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Validate all entries first before modifying registry
	for _, entry := range entries {
		name := entry.Name
		rt := entry.Type

		// If name is empty, infer it from the type
		if name == "" {
			name = inferTypeName(rt)
		}

		if !nameRegex.MatchString(name) {
			return fmt.Errorf("%w: invalid name %q", ErrTypeNotValid, name)
		}

		if rt.Kind() != reflect.Ptr || rt.Elem().Kind() != reflect.Struct {
			return fmt.Errorf("%w: %s must be a pointer to a struct", ErrTypeNotValid, rt)
		}

		if _, exists := r.types[name]; exists {
			return fmt.Errorf("%w: %s", ErrTypeAlreadyExists, name)
		}
	}

	// All validations passed, now register all types
	for _, entry := range entries {
		name := entry.Name
		rt := entry.Type

		if name == "" {
			name = inferTypeName(rt)
		}

		info := &TypeInfo{
			Type:     rt,
			Metadata: entry.Metadata,
			Validate: entry.Validate,
			Aliases:  []string{},
		}

		r.types[name] = info
		r.rtypes[normalizeType(rt)] = name
	}

	return nil
}

// --- Lookup --------------------------------------------------------

// resolveName resolves a name or alias to the primary type name
func (r *Registry) resolveName(name string) string {
	if primary, isAlias := r.aliases[name]; isAlias {
		return primary
	}
	return name
}

func (r *Registry) New(name string) (any, error) {
	if r == nil {
		return nil, fmt.Errorf("typeregistry: nil registry")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	// Resolve alias if necessary
	name = r.resolveName(name)

	info, ok := r.types[name]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrTypeNotRegistered, name)
	}

	return reflect.New(info.Type.Elem()).Interface(), nil
}

func (r *Registry) NameOf(v any) (string, error) {
	if r == nil {
		return "", fmt.Errorf("typeregistry: nil registry")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	if v == nil {
		return "", fmt.Errorf("%w: nil value", ErrTypeNotRegistered)
	}

	rt := normalizeType(reflect.TypeOf(v))
	name, ok := r.rtypes[rt]
	if !ok {
		return "", fmt.Errorf("%w: type not registered", ErrTypeNotRegistered)
	}
	return name, nil
}

func (r *Registry) Registered() []string {
	if r == nil {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.types))
	for n := range r.types {
		names = append(names, n)
	}
	return names
}

// Unregister removes a type from the registry, cleaning up both
// name and reverse type mappings to prevent memory leaks.
func (r *Registry) Unregister(name string) error {
	if r == nil {
		return fmt.Errorf("typeregistry: nil registry")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Resolve alias if necessary
	name = r.resolveName(name)

	info, ok := r.types[name]
	if !ok {
		return fmt.Errorf("%w: %s", ErrTypeNotRegistered, name)
	}

	// Remove all aliases for this type
	for _, alias := range info.Aliases {
		delete(r.aliases, alias)
	}

	delete(r.types, name)
	delete(r.rtypes, normalizeType(info.Type))

	// Clear any cached JSON schemas for this type
	r.jsonCache.Delete(name)

	return nil
}

// Clear removes all registered types.
func (r *Registry) Clear() {
	if r == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.types = make(map[string]*TypeInfo)
	r.rtypes = make(map[reflect.Type]string)
	r.aliases = make(map[string]string)

	// Clear all cached JSON schemas
	r.jsonCache = sync.Map{}
}

// --- JSON Helpers --------------------------------------------------

func (r *Registry) Marshal(v any) ([]byte, error) {
	if r == nil {
		return nil, fmt.Errorf("typeregistry: nil registry")
	}

	name, err := r.NameOf(v)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMarshal, err)
	}

	typed := &TypedData{
		Type: name,
		Data: data,
	}

	return json.Marshal(typed)
}

func (r *Registry) UnmarshalType(name string, data []byte) (any, error) {
	if r == nil {
		return nil, fmt.Errorf("typeregistry: nil registry")
	}

	r.mu.RLock()
	// Resolve alias if necessary
	name = r.resolveName(name)
	info, ok := r.types[name]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrTypeNotRegistered, name)
	}

	v := reflect.New(info.Type.Elem()).Interface()

	if err := json.Unmarshal(data, v); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnmarshal, err)
	}

	// Apply validation if configured
	if info.Validate != nil {
		if err := info.Validate(v); err != nil {
			return nil, fmt.Errorf("%w: validation failed: %v", ErrUnmarshal, err)
		}
	}

	return v, nil
}

func (r *Registry) Unmarshal(b []byte) (any, error) {
	if r == nil {
		return nil, fmt.Errorf("typeregistry: nil registry")
	}

	var typed TypedData

	if err := json.Unmarshal(b, &typed); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnmarshal, err)
	}

	if typed.Type == "" {
		return nil, fmt.Errorf("%w: missing type field", ErrUnmarshal)
	}

	return r.UnmarshalType(typed.Type, typed.Data)
}

// MarshalTypedData creates a TypedData structure from a registered value
func (r *Registry) MarshalTypedData(v any) (*TypedData, error) {
	if r == nil {
		return nil, fmt.Errorf("typeregistry: nil registry")
	}

	name, err := r.NameOf(v)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMarshal, err)
	}

	return &TypedData{
		Type: name,
		Data: data,
	}, nil
}

// UnmarshalTypedData unmarshals a TypedData structure into its registered type
func (r *Registry) UnmarshalTypedData(td *TypedData) (any, error) {
	if r == nil {
		return nil, fmt.Errorf("typeregistry: nil registry")
	}

	if td == nil {
		return nil, fmt.Errorf("%w: nil TypedData", ErrUnmarshal)
	}

	if td.Type == "" {
		return nil, fmt.Errorf("%w: missing type field", ErrUnmarshal)
	}

	return r.UnmarshalType(td.Type, td.Data)
}

// GetTypeInfo returns the TypeInfo for a registered type
func (r *Registry) GetTypeInfo(name string) (*TypeInfo, error) {
	if r == nil {
		return nil, fmt.Errorf("typeregistry: nil registry")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	// Resolve alias if necessary
	name = r.resolveName(name)

	info, ok := r.types[name]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrTypeNotRegistered, name)
	}

	return info, nil
}

// GetMetadata returns metadata for a registered type
func (r *Registry) GetMetadata(name string) (map[string]interface{}, error) {
	info, err := r.GetTypeInfo(name)
	if err != nil {
		return nil, err
	}
	return info.Metadata, nil
}

// FindByNamespace returns all types matching the given namespace prefix
func (r *Registry) FindByNamespace(namespace string) []string {
	if r == nil {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var names []string
	prefix := namespace + "."

	for name := range r.types {
		if strings.HasPrefix(name, prefix) || name == namespace {
			names = append(names, name)
		}
	}

	// Also check aliases
	for alias := range r.aliases {
		if strings.HasPrefix(alias, prefix) || alias == namespace {
			// Add the alias, not the primary name
			names = append(names, alias)
		}
	}

	return names
}
