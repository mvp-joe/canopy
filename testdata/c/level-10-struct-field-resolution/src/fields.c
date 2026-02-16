struct Rect {
    int width;
    int height;
};

struct Circle {
    double radius;
};

int area_rect(struct Rect *r) {
    return r->width * r->height;
}

double scale_radius(struct Circle c, double factor) {
    return c.radius * factor;
}
