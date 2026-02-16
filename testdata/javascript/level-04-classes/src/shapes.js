class Shape {
  name = "unknown";

  constructor(name) {
    this.name = name;
  }

  describe() {
    return this.name;
  }
}

class Circle extends Shape {
  radius = 0;

  constructor(radius) {
    super("circle");
    this.radius = radius;
  }

  area() {
    return Math.PI * this.radius * this.radius;
  }
}

function main() {
  const s = new Shape("square");
  const c = new Circle(5);
  console.log(s.describe());
  console.log(c.area());
}
