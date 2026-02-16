type Meters = f64;
type Kilometers = f64;
type Point = (f64, f64);

fn distance(a: Point, b: Point) -> Meters {
    let dx = a.0 - b.0;
    let dy = a.1 - b.1;
    (dx * dx + dy * dy).sqrt()
}

fn to_km(m: Meters) -> Kilometers {
    m / 1000.0
}

fn main() {
    let p1: Point = (0.0, 0.0);
    let p2: Point = (3.0, 4.0);
    let d = distance(p1, p2);
    let km = to_km(d);
}
