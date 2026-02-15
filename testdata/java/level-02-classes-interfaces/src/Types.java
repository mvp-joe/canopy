public interface Greeter {
    String greet(String name);
    void reset();
}

class SimpleGreeter implements Greeter {
    private String prefix;

    public SimpleGreeter(String prefix) {
        this.prefix = prefix;
    }

    public String greet(String name) {
        return prefix + " " + name;
    }

    public void reset() {
        this.prefix = "Hello";
    }

    public static Greeter create() {
        return new SimpleGreeter("Hi");
    }
}
