pub struct Product {
    pub id: u64,
    pub name: String,
    pub price: f64,
    pub active: bool,
}

impl Product {
    pub fn new(id: u64, name: String, price: f64) -> Self {
        Product {
            id,
            name,
            price,
            active: true,
        }
    }

    pub fn display_price(&self) -> String {
        format!("${:.2}", self.price)
    }

    pub fn deactivate(&mut self) {
        self.active = false;
    }
}

pub trait Displayable {
    fn summary(&self) -> String;
}

impl Displayable for Product {
    fn summary(&self) -> String {
        format!("{} ({})", self.name, self.display_price())
    }
}
