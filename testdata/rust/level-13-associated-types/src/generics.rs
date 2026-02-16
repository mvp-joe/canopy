pub trait Processor {
    fn process(&self, data: &str) -> String;
}

pub struct Upper;

impl Processor for Upper {
    fn process(&self, data: &str) -> String {
        data.to_uppercase()
    }
}

pub struct Lower;

impl Processor for Lower {
    fn process(&self, data: &str) -> String {
        data.to_lowercase()
    }
}

pub fn transform<P: Processor>(proc: &P, input: &str) -> String {
    proc.process(input)
}

pub fn transform_where<P>(proc: &P, input: &str) -> String
where
    P: Processor,
{
    proc.process(input)
}

fn main() {
    let u = Upper;
    let l = Lower;
    let r1 = transform(&u, "hello");
    let r2 = transform_where(&l, "WORLD");
}
