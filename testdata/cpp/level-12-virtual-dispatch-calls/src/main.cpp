#include "shapes.hpp"

void print_area(Shape& s) {
    s.area();
    s.describe();
}

int main() {
    Circle c;
    Square sq;
    print_area(c);
    print_area(sq);
    return 0;
}
