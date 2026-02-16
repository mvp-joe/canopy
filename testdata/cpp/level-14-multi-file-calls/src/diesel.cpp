#include "engine.hpp"

class Diesel : public Engine {
public:
    void start() override {}
    void stop() override {}
};

void test_diesel() {
    Diesel d;
    run_engine(d);
}
