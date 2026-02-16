def make_greeter(greeting):
    def greet(name):
        return greeting + ", " + name
    return greet


def make_adder(base):
    def add(x):
        return base + x
    return add


def pipeline():
    hello = make_greeter("Hello")
    plus_ten = make_adder(10)
    return hello("world"), plus_ten(5)
