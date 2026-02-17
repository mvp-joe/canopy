#include <stdio.h>

#define MAX_RETRIES 3

int debug = 0;

char* hello(void) {
    return "hello";
}

int add(int a, int b) {
    return a + b;
}

int main(void) {
    printf("%s\n", hello());
    int result = add(1, 2);
    return 0;
}
