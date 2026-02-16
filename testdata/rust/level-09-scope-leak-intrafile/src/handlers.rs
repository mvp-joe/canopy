struct Response {
    code: i32,
}

impl Response {
    fn new(code: i32) -> Response {
        Response { code }
    }
}

fn handle_a() -> Response {
    Response::new(200)
}

fn handle_b() -> Response {
    Response::new(404)
}
