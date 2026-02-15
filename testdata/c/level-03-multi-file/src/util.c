#include "util.h"

int square(int n) {
    return n * n;
}

int cube(int n) {
    int s = square(n);
    return s * n;
}
