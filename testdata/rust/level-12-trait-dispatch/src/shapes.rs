pub struct Circle {
    pub radius: f64,
}

impl Circle {
    pub fn new(radius: f64) -> Circle {
        Circle { radius }
    }
}

impl Drawable for Circle {
    fn draw(&self) -> String {
        "circle".to_string()
    }

    fn area(&self) -> f64 {
        3.14 * self.radius * self.radius
    }
}

pub struct Square {
    pub side: f64,
}

impl Drawable for Square {
    fn draw(&self) -> String {
        "square".to_string()
    }

    fn area(&self) -> f64 {
        self.side * self.side
    }
}

fn render(c: &Circle, s: &Square) {
    let cd = c.draw();
    let ca = c.area();
    let sd = s.draw();
    let sa = s.area();
}

fn main() {
    let c = Circle::new(5.0);
    render(&c, &c);
}
