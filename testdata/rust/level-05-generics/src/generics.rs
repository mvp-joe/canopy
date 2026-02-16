struct Pair<T> {
    first: T,
    second: T,
}

impl<T> Pair<T> {
    fn new(first: T, second: T) -> Self {
        Pair { first, second }
    }

    fn first(&self) -> &T {
        &self.first
    }
}

trait Summarize {
    fn summary(&self) -> String;
}

fn largest<T: PartialOrd>(a: T, b: T) -> T {
    if a > b { a } else { b }
}

fn print_it<T: Summarize>(item: &T) {
    println!("{}", item.summary());
}

fn main() {
    let p = Pair::new(10, 20);
    let big = largest(10, 20);
}
