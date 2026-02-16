class Point:
    def __init__(self, x: float, y: float):
        self.x = x
        self.y = y


def distance(a: Point, b: Point) -> float:
    dx = a.x - b.x
    dy = a.y - b.y
    return (dx * dx + dy * dy) ** 0.5


def origin() -> Point:
    return Point(0.0, 0.0)
