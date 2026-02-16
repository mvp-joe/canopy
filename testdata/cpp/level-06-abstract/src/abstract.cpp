#include <string>

class Shape {
public:
    virtual double area() = 0;
    virtual std::string name() = 0;
    virtual ~Shape() {}
};

class Printable {
public:
    virtual void print() = 0;
};

class Circle : public Shape, public Printable {
public:
    double radius;
    Circle(double r) : radius(r) {}
    double area() override { return 3.14 * radius * radius; }
    std::string name() override { return "Circle"; }
    void print() override {}
};
