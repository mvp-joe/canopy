class Animal {
public:
    virtual void speak() = 0;
    virtual int legs() { return 4; }
};

class Dog : public Animal {
public:
    void speak() override {}
    int legs() override { return 4; }
};

class Snake : public Animal {
public:
    void speak() override {}
    int legs() override { return 0; }
};
