mod models;
mod service;

use models::Product;
use service::ProductService;

fn main() {
    let mut svc = ProductService::new();
    let p = svc.add_product("Widget".to_string(), 9.99);
    println!("Added: {} at ${:.2}", p.name, p.price);

    let found = svc.find_by_name("Widget");
    match found {
        Some(product) => println!("Found: {}", product.name),
        None => println!("Not found"),
    }

    let all = svc.list_products();
    println!("Total products: {}", all.len());
}
