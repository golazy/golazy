package generator

type Option struct {
	Id                           string
	LongFlag, ShortFlag, EnvName string
	Default                      string
	Description                  string
}

type Command struct {
	Base
	Command string
}

func (c *Command) Generate(path string, vars map[string]string) error {

	return nil
}
