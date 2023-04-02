package generator

type Fn struct {
	Base
	WithVars
	Func func(path string, vars map[string]string) error
}

func (f *Fn) Generate(path string, vars map[string]string) error {
	if err := f.ValidateVars(vars); err != nil {
		return err
	}

	f.Func(path, vars)
	return nil
}
