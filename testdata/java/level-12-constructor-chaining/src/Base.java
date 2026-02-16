package chain;

public class Base {
    private String label;

    public Base(String label) {
        this.label = label;
    }

    public Base() {
        this("default");
    }

    public String getLabel() {
        return label;
    }
}
