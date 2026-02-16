package enums

type Color int

const (
	Red   Color = iota
	Green
	Blue
)

type LogLevel int

const (
	Debug LogLevel = iota
	Info
	Warn
	Error
)

func (c Color) String() string {
	return "color"
}
