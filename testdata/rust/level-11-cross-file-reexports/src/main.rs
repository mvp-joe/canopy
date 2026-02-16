use types::Config;
use types::default_name;

fn build() -> Config {
    let name = default_name();
    Config { name: name.to_string() }
}

fn main() {
    let cfg = build();
}
