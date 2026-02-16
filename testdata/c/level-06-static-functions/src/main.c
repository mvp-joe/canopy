#include <stdio.h>

int transform(int x);

static int double_val(int x) {
    return x + x;
}

int main(void) {
    int a = transform(7);
    int b = double_val(3);
    printf("a=%d b=%d\n", a, b);
    return 0;
}
