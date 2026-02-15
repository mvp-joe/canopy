#include <string>

const int MAX_RETRIES = 3;

bool debug = false;

std::string hello() {
    return "hello";
}

int add(int a, int b) {
    return a + b;
}

int main() {
    hello();
    int result = add(1, 2);
    return 0;
}
