package rest

import (
	"testing"

	"golazy.dev/lazyapp/apptest"
)

type User struct {
	Name string
	Age  string
}

type ValuesController struct {
	Controller
}

var UserStorage = NewMemStore[User]()

var TestController = &ValuesController{
	Controller: NewController(UserStorage),
}

type TestStore struct {
}

func (t *TestStore) List() ([]any, error) {
	return nil, nil
}

func (t *TestStore) Get(id string) (any, error) {
	return nil, nil
}

func (t *TestStore) Set(id string, v any) error {
	return nil
}
func (t *TestStore) Delete(key string) error {
	return nil
}

func TestRestController(t *testing.T) {

	expect := apptest.NewController(t, TestController).Expect

	expect("/").Code(200).Body(`[]`)

}
