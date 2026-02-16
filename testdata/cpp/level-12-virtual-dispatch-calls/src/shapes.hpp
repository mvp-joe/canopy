#pragma once

class Shape {
public:
    virtual double area() = 0;
    virtual void describe() {}
};

class Circle : public Shape {
public:
    double area() override { return 3.14; }
    void describe() override {}
};

class Square : public Shape {
public:
    double area() override { return 1.0; }
};
