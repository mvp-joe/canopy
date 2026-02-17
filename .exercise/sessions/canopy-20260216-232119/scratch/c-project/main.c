#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "stack.h"

int fibonacci(int n) {
    if (n <= 1) return n;
    return fibonacci(n - 1) + fibonacci(n - 2);
}

void print_array(int *arr, int size) {
    for (int i = 0; i < size; i++) {
        printf("%d ", arr[i]);
    }
    printf("\n");
}

int compare_ints(const void *a, const void *b) {
    return (*(int *)a - *(int *)b);
}

int main() {
    Stack *s = stack_create(10);
    stack_push(s, 42);
    stack_push(s, 17);
    stack_push(s, 99);

    printf("Top: %d\n", stack_peek(s));
    printf("Size: %d\n", stack_size(s));

    int val = stack_pop(s);
    printf("Popped: %d\n", val);

    printf("Fibonacci(10) = %d\n", fibonacci(10));

    int arr[] = {5, 2, 8, 1, 9};
    qsort(arr, 5, sizeof(int), compare_ints);
    print_array(arr, 5);

    stack_destroy(s);
    return 0;
}
