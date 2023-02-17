package lazyaction

type ResourceDefinition struct {
	Controller     any
	ControllerName string // PostsController
	Plural         string // posts
	Singular       string // post
	PathNames      struct{ New, Edit string }
	Path           string // "" means default, "/" means empty
	ParamName      string
	SubResources   []*ResourceDefinition
}
