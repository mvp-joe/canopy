#define VERSION_MAJOR 2
#define VERSION_MINOR 1
#define VERSION_PATCH 0

#define SQUARE(x) ((x) * (x))
#define MIN(a, b) ((a) < (b) ? (a) : (b))
#define MAX(a, b) ((a) > (b) ? (a) : (b))
#define CLAMP(x, lo, hi) MIN(MAX(x, lo), hi)

#define ARRAY_SIZE(arr) (sizeof(arr) / sizeof((arr)[0]))

int sum_squares(int n) {
    int total = 0;
    for (int i = 1; i <= n; i++) {
        total += SQUARE(i);
    }
    return total;
}
