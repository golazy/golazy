package command

var Commands = []Command{}

func Add(cmd Command) {
	Commands = append(Commands, cmd)
}

type Options struct {
	CurrentDir string
	MainDir    string
	AppDir     string
	Args       []string
	SubArgs    []string
	flags      map[string]any
	command    *Command
}

type Flag[T any] struct {
	Short       string
	Name        string
	Env         string
	Default     any
	Value       any
	Description string
}

type Command struct {
	Use       string
	Short     string
	Long      string
	Flags     []Flag
	Run       func(Options) error
	ValidArgs func(Options) []string
	MinArgs   int
	MaxArgs   int
}
