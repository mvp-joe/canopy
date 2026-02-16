enum Status {
    STATUS_OK,
    STATUS_ERROR,
    STATUS_PENDING
};

int is_done(enum Status s) {
    return s == STATUS_OK || s == STATUS_ERROR;
}

enum Status get_default(void) {
    return STATUS_PENDING;
}
