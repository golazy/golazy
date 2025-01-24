package lazycontext

import (
	"context"
	"fmt"
	"io"
	"testing"
)

func TestAppContextIsPresent(t *testing.T) {

	lctx2 := Get[AppContext](New())
	if lctx2 == nil {
		t.Error("context is not present in New()")
	}

	lctx2 = Get[AppContext](NewWithContext(context.Background()))
	if lctx2 == nil {
		t.Error("context is not present in NewWithContext()")
	}
}

func TestGetSetAppContext(t *testing.T) {
	ctx := New()
	lctx := Get[AppContext](ctx)
	if lctx == nil {
		t.Error("expected *lazycontext.AppContext, got nil")
	}
	_, ok := lctx.Value(KeyForType[AppContext](nil)).(AppContext)
	if !ok {
		t.Errorf("expected *lazycontext.AppContext")
	}
	type myStr struct {
	}
	Set(ctx, myStr{})
}

func TestDoubleAppContext(t *testing.T) {
	ctx := New()
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic, got nil")
		}
	}()
	Set(ctx, New())
}

func TestContext(t *testing.T) {

	ctx := New()

	Set(ctx, "test")
	value := Get[string](ctx)

	if value != "test" {
		t.Errorf("Expected 'test', got %v", value)
	}
}

type testStruct struct {
	Name string
}

func (ts testStruct) String() string {
	return ts.Name
}

func TestGetSet(t *testing.T) {
	ctx := New()

	// Works with struct
	ts := testStruct{Name: "test"}

	Set(ctx, ts)
	ts2 := Get[testStruct](ctx)
	if ts2.Name != ts.Name {
		t.Errorf("Expected %v, got %v", ts.Name, ts2.Name)
	}

	// Works with struct pointers
	ts3 := &testStruct{Name: "test2"}
	Set(ctx, ts3)
	ts4 := Get[*testStruct](ctx)
	if ts4.Name != ts3.Name {
		t.Errorf("Expected %v, got %v", ts3.Name, ts4.Name)
	}

	// Works with interfaces
	intfc := fmt.Stringer(ts)

	Set(ctx, intfc)

	intfc2 := Get[fmt.Stringer](ctx)
	if intfc2.String() != ts.Name {
		t.Errorf("Expected %v, got %v", ts.Name, intfc2.String())
	}

	// Works with missing values
	ctx = New()
	if Get[io.Reader](ctx) != nil {
		t.Errorf("Expected nil, got %v", Get[io.Reader](ctx))
	}
	if Get[testStruct](ctx).Name != "" {
		t.Errorf("Expected nil, got %v", Get[io.Reader](ctx))
	}
	if val := Get[*testStruct](ctx); val != nil {
		t.Errorf("Expected nil, got %T %v", val, val)
	}

}

func ExampleContext() {
	ctx := New()

	// You can store values as with normal context
	var userKey string
	ctx.AddValue(userKey, "user_33")
	fmt.Println(ctx.Value(userKey))

	// Or if you need to reference only by specific type you can omit the key
	type myConfig struct{ Name string }

	Set(ctx, myConfig{Name: "test"})

	cfg := Get[myConfig](ctx)
	fmt.Println(cfg.Name)

	// Output:
	// user_33
	// test

}
