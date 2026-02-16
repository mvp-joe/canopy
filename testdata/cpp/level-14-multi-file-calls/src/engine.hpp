#pragma once

class Engine {
public:
    virtual void start() = 0;
    virtual void stop() = 0;
};

void run_engine(Engine& e);
