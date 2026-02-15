struct Config {
    name: String,
    retries: u32,
}

trait Handler {
    fn handle(&self) -> bool;
    fn reset(&mut self);
}

impl Handler for Config {
    fn handle(&self) -> bool {
        true
    }

    fn reset(&mut self) {
        self.retries = 0;
    }
}

impl Config {
    fn new(name: String) -> Self {
        Config { name, retries: 3 }
    }
}
