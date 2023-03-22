//go:build production
// +build production

package lazydev

func init() {

	DefaultServer = &server{
		BootMode:  ProductionMode,
		HTTPAddr:  ":80",
		HTTPSAddr: ":443",
	}

}
