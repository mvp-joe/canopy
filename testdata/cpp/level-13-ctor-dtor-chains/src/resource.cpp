class Resource {
public:
    Resource() {}
    virtual ~Resource() {}
    void init() {}
    void cleanup() {}
};

class File : public Resource {
public:
    File() {}
    ~File() {}

    void open() {
        init();
    }
    void close() {
        cleanup();
    }
};

void process() {
    File f;
    f.open();
    f.close();
}
