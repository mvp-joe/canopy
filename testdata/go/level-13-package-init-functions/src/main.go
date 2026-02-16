package startup

var Ready bool

func setup() {
	Ready = true
}

func init() {
	setup()
}

func Run() int {
	return DefaultPort
}
