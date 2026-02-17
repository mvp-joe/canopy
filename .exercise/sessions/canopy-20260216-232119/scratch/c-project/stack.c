#include <stdlib.h>
#include "stack.h"

struct Stack {
    int *data;
    int top;
    int capacity;
};

Stack *stack_create(int capacity) {
    Stack *s = malloc(sizeof(Stack));
    s->data = malloc(sizeof(int) * capacity);
    s->top = -1;
    s->capacity = capacity;
    return s;
}

void stack_destroy(Stack *s) {
    free(s->data);
    free(s);
}

void stack_push(Stack *s, int value) {
    if (s->top < s->capacity - 1) {
        s->data[++s->top] = value;
    }
}

int stack_pop(Stack *s) {
    if (s->top >= 0) {
        return s->data[s->top--];
    }
    return -1;
}

int stack_peek(Stack *s) {
    if (s->top >= 0) {
        return s->data[s->top];
    }
    return -1;
}

int stack_size(Stack *s) {
    return s->top + 1;
}

int stack_is_empty(Stack *s) {
    return s->top < 0;
}
