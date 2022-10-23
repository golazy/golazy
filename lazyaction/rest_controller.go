package lazyaction

import (
	"net/http"
)

type MemStore[T any] map[string]*T

func NewMemStore[T any]() Storage[T] {
	return make(MemStore[T])
}

func (m MemStore[T]) List(params ...any) []*T {
	data := make([]*T, 0, len(m))
	for _, v := range m {
		data = append(data, v)
	}

	return data
}
func (m MemStore[T]) Read(key string) *T {
	return m[key]
}

func (m MemStore[T]) Destroy(key string) {
	delete(m, key)
}
func (m MemStore[T]) Write(key string, v *T) {
	m[key] = v
}

type Storage[T any] interface {
	List(...any) []*T
	Read(string) *T
	Destroy(string)
	Write(string, *T)
}

type RestController[T any] struct {
	S Storage[T]
}

func (rc *RestController[T]) Index(w http.ResponseWriter, r *http.Request) {

}

func (rc *RestController[T]) Show() {

}

func (rc *RestController[T]) Create() {

}

func (rc *RestController[T]) Update() {

}

func (rc *RestController[T]) Destroy() {

}

func (rc *RestController[T]) New() {

}
func (rc *RestController[T]) Edit() {

}
