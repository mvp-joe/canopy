const MAX_RETRIES: u32 = 3;

static DEBUG: bool = false;

fn hello() -> &'static str {
    "hello"
}

fn add(a: i32, b: i32) -> i32 {
    a + b
}

fn main() {
    println!("{}", hello());
    let result = add(1, 2);
}
