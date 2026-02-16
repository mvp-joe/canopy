package startup

var DefaultPort int

func SetPort(p int) {
	DefaultPort = p
}

func init() {
	SetPort(8080)
}
