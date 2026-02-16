#include <stdio.h>

typedef int (*compare_fn)(int, int);

int ascending(int a, int b) {
    return a - b;
}

int descending(int a, int b) {
    return b - a;
}

void sort_pair(int *x, int *y, compare_fn cmp) {
    if (cmp(*x, *y) > 0) {
        int tmp = *x;
        *x = *y;
        *y = tmp;
    }
}

int main(void) {
    int a = 5, b = 3;
    sort_pair(&a, &b, ascending);
    printf("%d %d\n", a, b);
    return 0;
}
