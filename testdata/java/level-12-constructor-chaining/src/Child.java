package chain;

public class Child extends Base {
    private int value;

    public Child(String label, int value) {
        super(label);
        this.value = value;
    }

    public Child(int value) {
        this("child", value);
    }

    public int getValue() {
        return value;
    }
}
