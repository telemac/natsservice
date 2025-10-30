package typeregistry

import (
	"encoding/json"
	"errors"
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

type User struct {
	Name string
	Age  int
}

type Order struct {
	ID string
}

func newRegistry(t *testing.T) *Registry {
	r := New()
	MustRegister[User](r, "example.user")
	MustRegister[Order](r, "example.order")
	MustRegister[Order](r, "")
	return r
}

func TestRegisterAndLookup(t *testing.T) {
	assert := assert.New(t)
	r := newRegistry(t)

	names := r.Registered()
	assert.Contains(names, "example.user")
	assert.Contains(names, "example.order")

	name, err := r.NameOf(&User{})
	assert.NoError(err)
	assert.Equal("example.user", name)
}

func TestNew(t *testing.T) {
	assert := assert.New(t)
	r := newRegistry(t)

	v, err := r.New("example.user")
	assert.NoError(err)

	u, ok := v.(*User)
	assert.True(ok)
	assert.NotNil(u)
}

func TestMarshalAndUnmarshal(t *testing.T) {
	assert := assert.New(t)
	r := newRegistry(t)

	u := &User{Name: "Alexandre", Age: 33}
	data, err := r.Marshal(u)
	assert.NoError(err)

	var m map[string]json.RawMessage
	assert.NoError(json.Unmarshal(data, &m))
	assert.Contains(m, "type")
	assert.Contains(m, "data")

	v, err := r.Unmarshal(data)
	assert.NoError(err)

	u2, ok := v.(*User)
	assert.True(ok)
	assert.Equal(u.Name, u2.Name)
	assert.Equal(u.Age, u2.Age)
}

func TestUnmarshalType(t *testing.T) {
	assert := assert.New(t)
	r := newRegistry(t)

	jsonValue := []byte(`{"Name":"Alexandre","Age":33}`)
	v, err := r.UnmarshalType("example.user", jsonValue)
	assert.NoError(err)

	u := v.(*User)
	assert.Equal("Alexandre", u.Name)
	assert.Equal(33, u.Age)
}

func TestUnknownType(t *testing.T) {
	assert := assert.New(t)
	r := newRegistry(t)

	_, err := r.New("does.not.exist")
	assert.ErrorIs(err, ErrTypeNotRegistered)

	_, err = r.Unmarshal([]byte(`{"type":"does.not.exist","data":{}}`))
	assert.ErrorIs(err, ErrTypeNotRegistered)
}

func TestNilRegistry(t *testing.T) {
	assert := assert.New(t)
	var r *Registry

	_, err := r.New("test")
	assert.Error(err)
	assert.Contains(err.Error(), "nil registry")

	_, err = r.NameOf(&User{})
	assert.Error(err)
	assert.Contains(err.Error(), "nil registry")

	assert.Nil(r.Registered())

	_, err = r.UnmarshalType("test", []byte("{}"))
	assert.Error(err)
	assert.Contains(err.Error(), "nil registry")

	_, err = r.Unmarshal([]byte("{}"))
	assert.Error(err)
	assert.Contains(err.Error(), "nil registry")

	_, err = r.Marshal(&User{})
	assert.Error(err)
	assert.Contains(err.Error(), "nil registry")
}

func TestNilValue(t *testing.T) {
	assert := assert.New(t)
	r := New()

	_, err := r.NameOf(nil)
	assert.ErrorIs(err, ErrTypeNotRegistered)
	assert.Contains(err.Error(), "nil value")
}

func TestUnregister(t *testing.T) {
	assert := assert.New(t)
	r := newRegistry(t)

	assert.Contains(r.Registered(), "example.user")

	err := r.Unregister("example.user")
	assert.NoError(err)
	assert.NotContains(r.Registered(), "example.user")

	_, err = r.New("example.user")
	assert.ErrorIs(err, ErrTypeNotRegistered)

	err = r.Unregister("example.user")
	assert.ErrorIs(err, ErrTypeNotRegistered)
}

func TestClear(t *testing.T) {
	assert := assert.New(t)
	r := newRegistry(t)

	assert.Greater(len(r.Registered()), 0)

	r.Clear()
	assert.Empty(r.Registered())

	_, err := r.New("example.user")
	assert.ErrorIs(err, ErrTypeNotRegistered)
}

func TestInvalidNames(t *testing.T) {
	assert := assert.New(t)
	r := New()

	// Empty name is now valid (auto-generated)
	err := Register[User](r, "")
	assert.NoError(err)

	// But invalid names with special characters are still rejected
	err = Register[User](r, "invalid name!")
	assert.ErrorIs(err, ErrTypeNotValid)
	assert.Contains(err.Error(), "invalid name")
}

func TestInvalidTypes(t *testing.T) {
	assert := assert.New(t)
	r := New()

	err := Register[string](r, "test.string")
	assert.ErrorIs(err, ErrTypeNotValid)
	assert.Contains(err.Error(), "pointer to a struct")

	err = Register[map[string]int](r, "test.map")
	assert.ErrorIs(err, ErrTypeNotValid)
	assert.Contains(err.Error(), "pointer to a struct")
}

func TestDuplicateRegistration(t *testing.T) {
	assert := assert.New(t)
	r := New()

	err := Register[User](r, "duplicate")
	assert.NoError(err)

	err = Register[Order](r, "duplicate")
	assert.ErrorIs(err, ErrTypeAlreadyExists)
}

func TestConcurrentAccess(t *testing.T) {
	assert := assert.New(t)
	r := New()
	MustRegister[User](r, "concurrent.user")

	var wg sync.WaitGroup
	numGoroutines := 100
	errors := make(chan error, numGoroutines*2)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, err := r.New("concurrent.user")
			if err != nil {
				errors <- err
			}
		}()
		go func() {
			defer wg.Done()
			_ = r.Registered()
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		assert.NoError(err)
	}
}

func TestMalformedJSON(t *testing.T) {
	assert := assert.New(t)
	r := New()

	_, err := r.Unmarshal([]byte("invalid json"))
	assert.Error(err)
	assert.Contains(err.Error(), "unmarshal error")

	_, err = r.Unmarshal([]byte(`{}`))
	assert.Error(err)
	assert.Contains(err.Error(), "missing type field")

	_, err = r.Unmarshal([]byte(`{"type":"unknown","data":{}}`))
	assert.ErrorIs(err, ErrTypeNotRegistered)
}

func TestAutoNaming(t *testing.T) {
	assert := assert.New(t)
	r := New()

	// Register types with empty names (auto-naming)
	err := Register[User](r, "")
	assert.NoError(err)

	err = Register[Order](r, "")
	assert.NoError(err)

	// Check that types were registered with auto-generated names
	// Since these types are in the same package, they should be registered as
	// "typeregistry.User" and "typeregistry.Order"
	names := r.Registered()
	assert.Contains(names, "typeregistry.User")
	assert.Contains(names, "typeregistry.Order")

	// Test creating instances with auto-generated names
	user, err := r.New("typeregistry.User")
	assert.NoError(err)
	assert.IsType(&User{}, user)

	order, err := r.New("typeregistry.Order")
	assert.NoError(err)
	assert.IsType(&Order{}, order)

	// Test NameOf with auto-registered types
	userInstance := &User{Name: "Test", Age: 25}
	name, err := r.NameOf(userInstance)
	assert.NoError(err)
	assert.Equal("typeregistry.User", name)
}

func TestAutoNamingWithCustomNames(t *testing.T) {
	assert := assert.New(t)
	r := New()

	// Mix auto-naming and custom naming
	err := Register[User](r, "")
	assert.NoError(err)

	err = Register[Order](r, "custom.order.name")
	assert.NoError(err)

	// Both should be registered
	names := r.Registered()
	assert.Contains(names, "typeregistry.User")
	assert.Contains(names, "custom.order.name")

	// Both should be creatable
	user, err := r.New("typeregistry.User")
	assert.NoError(err)
	assert.IsType(&User{}, user)

	order, err := r.New("custom.order.name")
	assert.NoError(err)
	assert.IsType(&Order{}, order)
}

func TestTypeMetadata(t *testing.T) {
	assert := assert.New(t)
	r := New()

	// Register with metadata
	metadata := map[string]interface{}{
		"version": "1.0",
		"author":  "test",
	}
	err := RegisterWithMetadata[User](r, "user.v1", metadata)
	assert.NoError(err)

	// Retrieve metadata
	retrieved, err := r.GetMetadata("user.v1")
	assert.NoError(err)
	assert.Equal(metadata["version"], retrieved["version"])
	assert.Equal(metadata["author"], retrieved["author"])

	// Get TypeInfo directly
	info, err := r.GetTypeInfo("user.v1")
	assert.NoError(err)
	assert.NotNil(info)
	assert.Equal(metadata, info.Metadata)
}

func TestValidationHook(t *testing.T) {
	assert := assert.New(t)
	r := New()

	// Register with validation
	validate := func(v any) error {
		u, ok := v.(*User)
		if !ok {
			return errors.New("invalid type")
		}
		if u.Age < 0 {
			return errors.New("age must be positive")
		}
		return nil
	}

	err := RegisterWithValidation[User](r, "validated.user", validate)
	assert.NoError(err)

	// Test valid unmarshaling
	validJSON := []byte(`{"Name":"John","Age":25}`)
	v, err := r.UnmarshalType("validated.user", validJSON)
	assert.NoError(err)
	user := v.(*User)
	assert.Equal("John", user.Name)
	assert.Equal(25, user.Age)

	// Test invalid unmarshaling (negative age)
	invalidJSON := []byte(`{"Name":"Jane","Age":-5}`)
	_, err = r.UnmarshalType("validated.user", invalidJSON)
	assert.Error(err)
	assert.Contains(err.Error(), "age must be positive")
}

func TestTypeAliasing(t *testing.T) {
	assert := assert.New(t)
	r := New()

	// Register a type
	err := Register[User](r, "original.user")
	assert.NoError(err)

	// Add aliases
	err = r.AddAlias("user.v1", "original.user")
	assert.NoError(err)

	err = r.AddAlias("legacy.user", "original.user")
	assert.NoError(err)

	// All names should work for creating instances
	original, err := r.New("original.user")
	assert.NoError(err)
	assert.IsType(&User{}, original)

	v1, err := r.New("user.v1")
	assert.NoError(err)
	assert.IsType(&User{}, v1)

	legacy, err := r.New("legacy.user")
	assert.NoError(err)
	assert.IsType(&User{}, legacy)

	// TypeInfo should show aliases
	info, err := r.GetTypeInfo("original.user")
	assert.NoError(err)
	assert.Contains(info.Aliases, "user.v1")
	assert.Contains(info.Aliases, "legacy.user")

	// Cannot add duplicate alias
	err = r.AddAlias("user.v1", "original.user")
	assert.ErrorIs(err, ErrTypeAlreadyExists)

	// Cannot alias non-existent type
	err = r.AddAlias("new.alias", "non.existent")
	assert.ErrorIs(err, ErrTypeNotRegistered)
}

func TestBatchRegistration(t *testing.T) {
	assert := assert.New(t)
	r := New()

	var userType User
	var orderType Order

	entries := []TypeEntry{
		{
			Name: "batch.user",
			Type: reflect.TypeOf(&userType),
			Metadata: map[string]interface{}{
				"version": "1.0",
			},
		},
		{
			Name: "batch.order",
			Type: reflect.TypeOf(&orderType),
			Metadata: map[string]interface{}{
				"version": "2.0",
			},
		},
	}

	err := r.RegisterBatch(entries)
	assert.NoError(err)

	// Both types should be registered
	names := r.Registered()
	assert.Contains(names, "batch.user")
	assert.Contains(names, "batch.order")

	// Check metadata
	userMeta, err := r.GetMetadata("batch.user")
	assert.NoError(err)
	assert.Equal("1.0", userMeta["version"])

	orderMeta, err := r.GetMetadata("batch.order")
	assert.NoError(err)
	assert.Equal("2.0", orderMeta["version"])
}

func TestFindByNamespace(t *testing.T) {
	assert := assert.New(t)
	r := New()

	// Register types in different namespaces
	Register[User](r, "app.models.User")
	Register[Order](r, "app.models.Order")
	Register[User](r, "app.dto.UserDTO")
	Register[Order](r, "legacy.Order")

	// Add an alias in a namespace
	r.AddAlias("app.models.UserV2", "app.models.User")

	// Find by namespace
	models := r.FindByNamespace("app.models")
	assert.Len(models, 3)
	assert.Contains(models, "app.models.User")
	assert.Contains(models, "app.models.Order")
	assert.Contains(models, "app.models.UserV2")

	dto := r.FindByNamespace("app.dto")
	assert.Len(dto, 1)
	assert.Contains(dto, "app.dto.UserDTO")

	legacy := r.FindByNamespace("legacy")
	assert.Len(legacy, 1)
	assert.Contains(legacy, "legacy.Order")
}

func TestTypedData(t *testing.T) {
	assert := assert.New(t)
	r := New()

	// Register a type
	err := Register[User](r, "test.User")
	assert.NoError(err)

	user := &User{Name: "John", Age: 30}

	// Test MarshalTypedData
	typed, err := r.MarshalTypedData(user)
	assert.NoError(err)
	assert.NotNil(typed)
	assert.Equal("test.User", typed.Type)
	assert.NotEmpty(typed.Data)

	// Test UnmarshalTypedData
	result, err := r.UnmarshalTypedData(typed)
	assert.NoError(err)
	u, ok := result.(*User)
	assert.True(ok)
	assert.Equal("John", u.Name)
	assert.Equal(30, u.Age)

	// Test TypedData helper methods
	td := NewTypedData("test.User", nil)
	assert.Equal("test.User", td.Type)

	err = td.MarshalValue(user)
	assert.NoError(err)
	assert.NotEmpty(td.Data)

	var decoded User
	err = td.UnmarshalValue(&decoded)
	assert.NoError(err)
	assert.Equal("John", decoded.Name)
	assert.Equal(30, decoded.Age)

	// Test JSON serialization of TypedData
	jsonData, err := json.Marshal(typed)
	assert.NoError(err)

	var newTyped TypedData
	err = json.Unmarshal(jsonData, &newTyped)
	assert.NoError(err)
	assert.Equal(typed.Type, newTyped.Type)
	assert.Equal(typed.Data, newTyped.Data)
}
