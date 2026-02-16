#define PI 3.14159
#define SQUARE(x) ((x) * (x))

typedef int Integer;
typedef unsigned long Size;

Integer add(Integer a, Integer b) {
    return a + b;
}

Size compute() {
    return SQUARE(5);
}
