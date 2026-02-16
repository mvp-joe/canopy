pub struct Logger {
    pub prefix: String,
}

impl Logger {
    pub fn new(prefix: &str) -> Logger {
        Logger { prefix: prefix.to_string() }
    }

    pub fn log(&self, msg: &str) {
        println!("{}: {}", self.prefix, msg);
    }
}
