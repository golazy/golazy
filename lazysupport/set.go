package lazysupport

type Void struct{}

type Set[T comparable] map[T]Void

func NewSet[T comparable](values ...T) Set[T] {
	s := make(Set[T])
	for _, key := range values {
		s[key] = Void{}
	}
	return s
}

func (s Set[T]) Has(item T) bool {
	_, ok := s[item]
	return ok
}
func (s Set[T]) Set(item T) {
	s[item] = Void{}
}

func (s Set[T]) Slice() []T {
	out := make([]T, 0, len(s))
	for key := range s {
		out = append(out, key)
	}
	return out
}
