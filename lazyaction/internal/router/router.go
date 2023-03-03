package router

func NewRouter[T any]() Matcher[T] {
	return NewMethodMatcher[T]()
}
