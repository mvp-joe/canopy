package chain;

public class App {
    public static void main(String[] args) {
        Child c = new Child(42);
        String label = c.getLabel();
        int val = c.getValue();
        System.out.println(label + ": " + val);
    }
}
