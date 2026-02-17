#ifndef STACK_H
#define STACK_H

typedef struct Stack Stack;

Stack *stack_create(int capacity);
void stack_destroy(Stack *s);
void stack_push(Stack *s, int value);
int stack_pop(Stack *s);
int stack_peek(Stack *s);
int stack_size(Stack *s);
int stack_is_empty(Stack *s);

#endif
