use prelude::*;
use logger::Logger;

fn run() -> Logger {
    let msg = greet("world");
    let logger = Logger::new("app");
    logger.log(&msg);
    logger
}

fn main() {
    let l = run();
    l.log("done");
}
