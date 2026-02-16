struct Worker {
    id: i32,
}

impl Worker {
    fn new(id: i32) -> Worker {
        Worker { id }
    }

    fn process(&self) -> i32 {
        self.id * 2
    }
}
