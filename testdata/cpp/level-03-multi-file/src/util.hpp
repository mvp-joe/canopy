#pragma once

#include <string>

namespace util {

int clamp(int val, int lo, int hi);

std::string repeat(const std::string& s, int n);

struct Config {
    std::string name;
    int retries;
};

} // namespace util
