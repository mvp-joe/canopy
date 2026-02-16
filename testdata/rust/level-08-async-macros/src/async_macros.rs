macro_rules! vec_of_strings {
    ($($s:expr),*) => {
        vec![$($s.to_string()),*]
    };
}

async fn fetch(url: &str) -> String {
    url.to_string()
}

async fn process(data: String) -> usize {
    data.len()
}

fn sync_helper() -> bool {
    true
}

fn main() {
    let v = vec_of_strings!("hello", "world");
    let ok = sync_helper();
}
