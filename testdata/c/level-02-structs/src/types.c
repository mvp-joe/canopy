#include <stdio.h>
#include <string.h>

struct Point {
    int x;
    int y;
};

typedef struct Point Point;

enum Color {
    RED,
    GREEN,
    BLUE
};

Point make_point(int x, int y) {
    Point p;
    p.x = x;
    p.y = y;
    return p;
}

void print_point(Point p) {
    printf("(%d, %d)\n", p.x, p.y);
}
