package lazyaction

import (
	"encoding/json"
	"net/http"
)

type MemStore[T any] map[string]*T

func NewMemStore[T any]() Storage[T] {
	return make(MemStore[T])
}

func (m MemStore[T]) New() *T {
	return new(T)
}

func (m MemStore[T]) List(params ...any) ([]*T, error) {
	data := make([]*T, 0, len(m))
	for _, v := range m {
		data = append(data, v)
	}

	return data, nil
}
func (m MemStore[T]) Read(key string) (*T, error) {
	return m[key], nil
}

func (m MemStore[T]) Destroy(key string) error {
	delete(m, key)
	return nil
}
func (m MemStore[T]) Write(key string, v *T) error {
	m[key] = v
	return nil
}

type Storage[K any] interface {
	List(...any) ([]*K, error)
	Read(string) (*K, error)
	Destroy(string) error
	Write(string, *K) error
	New() *K
}

type RestController[T any, J Storage[T]] struct {
	S J
}

func (rc *RestController[T, J]) Index(w http.ResponseWriter, r *http.Request) error {
	rows, err := rc.S.List()
	if err != nil {
		return err
	}

	return json.NewEncoder(w).Encode(rows)
}

func (rc *RestController[T, J]) Show(w http.ResponseWriter, id string) error {
	r, err := rc.S.Read(id)
	if err != nil {
		return err
	}
	return json.NewEncoder(w).Encode(r)

}

func (rc *RestController[T, J]) Create() {
}

func (rc *RestController[T, J]) Update() {

}

func (rc *RestController[T, J]) Destroy() {

}

func (rc *RestController[T, J]) New() {

}
func (rc *RestController[T, J]) Edit() {

}
