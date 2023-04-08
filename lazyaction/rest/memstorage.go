package rest

type MemStore[T any] map[string]*T

func NewMemStore[T any]() MemStore[T] {
	return make(MemStore[T])
}

func (m MemStore[T]) New() *T {
	return new(T)
}

func (m MemStore[T]) List() ([]any, error) {
	data := make([]any, 0, len(m))
	for _, v := range m {
		data = append(data, v)
	}

	return data, nil
}
func (m MemStore[T]) Get(key string) (any, error) {
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
