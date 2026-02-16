package dispatch

func UseCounter() string {
	c := &Counter{count: 0}
	c.Increment()
	v := c.Value()
	_ = v
	return "done"
}

func UseLogger() string {
	l := &Logger{prefix: "app"}
	l.SetPrefix("new")
	return l.Info("hello")
}
