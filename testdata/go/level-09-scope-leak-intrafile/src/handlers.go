package demo

type Response struct {
	Code int
}

func (r *Response) String() string {
	return "response"
}

func HandleA() string {
	r := &Response{Code: 200}
	return r.String()
}

func HandleB() string {
	r := &Response{Code: 404}
	return r.String()
}
