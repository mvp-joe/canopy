package dispatch

type Counter struct {
	count int
}

func (c *Counter) Increment() {
	c.count++
}

func (c Counter) Value() int {
	return c.count
}

type Logger struct {
	prefix string
}

func (l Logger) Info(msg string) string {
	return l.prefix + ": " + msg
}

func (l *Logger) SetPrefix(p string) {
	l.prefix = p
}
