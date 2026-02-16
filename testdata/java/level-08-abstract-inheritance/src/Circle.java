public class Circle extends Shape {
    private double radius;

    public Circle(double radius) {
        super("red");
        this.radius = radius;
    }

    @Override
    public double area() {
        return Math.PI * radius * radius;
    }

    public double circumference() {
        return 2 * Math.PI * radius;
    }
}
