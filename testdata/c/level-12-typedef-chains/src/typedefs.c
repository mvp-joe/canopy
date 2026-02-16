struct Vec2 {
    float x;
    float y;
};

typedef struct Vec2 Vector2;
typedef Vector2 Point2D;

Point2D make_point(float x, float y) {
    Point2D p;
    p.x = x;
    p.y = y;
    return p;
}

Vector2 add(Vector2 a, Vector2 b) {
    Vector2 result;
    result.x = a.x + b.x;
    result.y = a.y + b.y;
    return result;
}
