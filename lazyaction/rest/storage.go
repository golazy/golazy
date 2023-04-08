package rest

type Storage interface {
	Name() string
	List() ([]any, error)
	Get(id string) (any, error)
	New() any
}

type StorageWriter interface {
	Set(id string, v any) error
}

type StorageEraser interface {
	Delete(key string) error
}
