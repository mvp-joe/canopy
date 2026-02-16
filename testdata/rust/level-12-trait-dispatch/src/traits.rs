pub trait Drawable {
    fn draw(&self) -> String;
    fn area(&self) -> f64;
}
