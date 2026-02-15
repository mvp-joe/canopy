#include <string>

class Animal {
public:
    std::string name;
    Animal(std::string n) : name(n) {}
    virtual std::string speak() { return "..."; }
};

class Dog : public Animal {
public:
    Dog(std::string n) : Animal(n) {}
    std::string speak() override { return "woof"; }
};

struct Point {
    double x;
    double y;
};

Animal* createAnimal(std::string name) {
    return new Dog(name);
}
