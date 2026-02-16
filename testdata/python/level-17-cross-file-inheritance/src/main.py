from shapes import Circle, Square


def report():
    c = Circle(5)
    s = Square(3)
    return c.describe(), s.describe()
