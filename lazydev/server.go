package lazydev

type bootMode int
type sslMode int

const (
	autoCerts sslMode = iota
	lego
)

const (
	ParentMode bootMode = iota
	ChildMode
	ProductionMode
)

type server struct {
	BootMode            bootMode
	HTTPAddr, HTTPSAddr string
	productionServer
}

var DefaultServer = &server{
	BootMode:  ParentMode,
	HTTPAddr:  ":3000",
	HTTPSAddr: ":3000",
}

func (s *server) IsProduction() bool {
	return s.BootMode == ProductionMode
}

func IsProduction() bool {
	return DefaultServer.IsProduction()
}
