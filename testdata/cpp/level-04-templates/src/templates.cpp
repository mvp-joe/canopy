#include <string>

template<typename T>
T max_val(T a, T b) {
    return (a > b) ? a : b;
}

template<typename T, typename U>
class Pair {
public:
    T first;
    U second;
    Pair(T f, U s) : first(f), second(s) {}
    T getFirst() { return first; }
};

int main() {
    int m = max_val(3, 5);
    Pair<int, std::string> p(1, "hello");
    return 0;
}
