package component

type GFont struct {
	Font   string
	ApiKey string
}

func (g *GFont) Install(opts InstallOptions) error {
	return nil
}
