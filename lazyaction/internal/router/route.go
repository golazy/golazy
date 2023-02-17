package router

import "fmt"

type Route[T any] struct {
	Verb           string
	Path           string
	Name           string
	Target         T
	ResourceName   string // comment or comments or post_comments
	ResourceMember bool   // true if it adds a member path
	ResourceAction string // "new" or "edit" or custom
	ParamsPosition []int
	Member         bool
}

func (rd *Route[T]) String() string {
	return fmt.Sprintf("%s %s %s %v", rd.Name, rd.Verb, rd.Path, rd.Target)
}
