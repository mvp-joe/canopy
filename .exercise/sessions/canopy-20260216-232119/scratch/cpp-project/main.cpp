#include <iostream>
#include <vector>
#include <string>
#include <algorithm>
#include "shapes.hpp"

int main() {
    std::vector<Shape*> shapes;
    shapes.push_back(new Circle(5.0));
    shapes.push_back(new Rectangle(4.0, 6.0));
    shapes.push_back(new Triangle(3.0, 4.0, 5.0));

    for (const auto& shape : shapes) {
        std::cout << shape->name() << ": area = " << shape->area()
                  << ", perimeter = " << shape->perimeter() << std::endl;
    }

    auto maxArea = std::max_element(shapes.begin(), shapes.end(),
        [](const Shape* a, const Shape* b) {
            return a->area() < b->area();
        });

    std::cout << "Largest shape: " << (*maxArea)->name() << std::endl;

    for (auto shape : shapes) {
        delete shape;
    }
    return 0;
}
