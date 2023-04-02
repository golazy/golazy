package lazyplug

type Generator interface {
	Name()
}

var Generators []Generator

func NewGenerator(g Generator) {
	Generators = append(Generators, g)
}
