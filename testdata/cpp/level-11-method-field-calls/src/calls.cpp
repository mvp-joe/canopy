class Logger {
public:
    void info(const char* msg) {}
    void warn(const char* msg) {}
};

void greet() {}

void process(Logger& log) {
    greet();
    log.info("starting");
    log.warn("careful");
}

int main() {
    Logger log;
    process(log);
    return 0;
}
