mod util;

fn main() {
    let msg = util::greet("world");
    println!("{}", msg);

    let sum = util::add(2, 3);
    println!("{}", sum);
}
