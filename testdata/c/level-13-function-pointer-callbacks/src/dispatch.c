typedef void (*handler_fn)(int);

struct Handlers {
    handler_fn on_start;
    handler_fn on_stop;
};

void start_handler(int code) {
}

void stop_handler(int code) {
}

void run_handlers(struct Handlers *h) {
    h->on_start(0);
    h->on_stop(1);
}

void setup(void) {
    struct Handlers h;
    h.on_start = start_handler;
    h.on_stop = stop_handler;
    run_handlers(&h);
}
