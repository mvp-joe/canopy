use crate::models::Product;

pub struct ProductService {
    products: Vec<Product>,
    next_id: u64,
}

impl ProductService {
    pub fn new() -> Self {
        ProductService {
            products: Vec::new(),
            next_id: 1,
        }
    }

    pub fn add_product(&mut self, name: String, price: f64) -> &Product {
        let product = Product::new(self.next_id, name, price);
        self.next_id += 1;
        self.products.push(product);
        self.products.last().unwrap()
    }

    pub fn find_by_name(&self, name: &str) -> Option<&Product> {
        self.products.iter().find(|p| p.name == name)
    }

    pub fn list_products(&self) -> &[Product] {
        &self.products
    }

    pub fn remove_product(&mut self, id: u64) -> bool {
        if let Some(pos) = self.products.iter().position(|p| p.id == id) {
            self.products.remove(pos);
            true
        } else {
            false
        }
    }
}
