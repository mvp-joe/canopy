package types

type Config struct {
	Host string
	Port int
}

type Handler interface {
	Handle(req string) (string, error)
	Close() error
}

type Server struct {
	Config
	Name string
}

func (s *Server) Handle(req string) (string, error) {
	return "ok", nil
}

func (s *Server) Close() error {
	return nil
}

func NewServer(name string) *Server {
	return &Server{Name: name}
}
