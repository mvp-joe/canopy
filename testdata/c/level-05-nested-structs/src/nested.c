struct Address {
    char street[100];
    char city[50];
    int zip;
};

struct Person {
    char name[50];
    int age;
    struct Address home;
    struct Address work;
};

struct Company {
    char name[100];
    struct Person ceo;
    int employee_count;
};

void init_address(struct Address *addr, int zip) {
    addr->zip = zip;
}

void init_person(struct Person *p, int age) {
    p->age = age;
    init_address(&p->home, 10001);
}
