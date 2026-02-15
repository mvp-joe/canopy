#include <stdio.h>
#include "util.h"

int main(void) {
    int a = square(5);
    int b = cube(3);
    printf("square=%d cube=%d\n", a, b);
    return 0;
}
