static int double_val(int x) {
    return x * 2;
}

static int clamp(int x, int lo, int hi) {
    if (x < lo) return lo;
    if (x > hi) return hi;
    return x;
}

int transform(int x) {
    return clamp(double_val(x), 0, 100);
}
