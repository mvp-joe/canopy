package embed

type Reader interface {
	Read(data string) int
}

type Writer interface {
	Write(data string) int
}

type ReadWriter interface {
	Reader
	Writer
}

type MyReader struct {
	Name string
}

func (r *MyReader) Read(data string) int {
	return len(data)
}

type MyReadWriter struct {
	MyReader
	Tag string
}

func (rw *MyReadWriter) Write(data string) int {
	return len(data)
}
