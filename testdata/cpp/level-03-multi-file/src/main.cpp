#include "util.hpp"
#include <iostream>

int main() {
    util::Config cfg{"app", 5};

    int val = util::clamp(10, 0, 100);

    std::string line = util::repeat("-", 20);

    std::cout << cfg.name << std::endl;
    std::cout << val << std::endl;
    std::cout << line << std::endl;

    return 0;
}
