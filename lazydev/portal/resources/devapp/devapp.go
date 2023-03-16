package devapp

type devApp struct {
	root     string
	bin      string
	status   Status
	buildOut []byte
	appOut   [][]byte
}

type Status int

const (
	maxOut = 1000
)

const (
	AppRuning Status = iota
	AppBuilding
	AppStopped
)

func (d *devApp) SetRoot(root string) {
	d.root = root
}

func (d *devApp) SetBuildOut(out []byte) {
	d.buildOut = out
}

func (d *devApp) SetBin(bin string) {
	d.bin = bin
}

func (d *devApp) AppendAppOut(out []byte) {
	d.appOut = append(d.appOut, out)
	if len(d.appOut) > maxOut {
		d.appOut = d.appOut[1:]
	}
}

func (d *devApp) SetStatus(status Status) {
	if status == d.status {
		return
	}
	switch status {
	case AppRuning:
		d.appOut = make([][]byte, 0, 100)
	case AppBuilding:
		d.buildOut = []byte{}
	case AppStopped:
	}
	d.status = status
}

func (d *devApp) Status() Status {
	return d.status
}

type Route struct {
}

func (d *devApp) Routes() []Route {

	return nil
}

func new() *devApp {

	return &devApp{}
}

var DevApp = new()
