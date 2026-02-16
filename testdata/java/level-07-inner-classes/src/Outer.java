public class Outer {
    private int value;

    public Outer(int value) {
        this.value = value;
    }

    public static class StaticNested {
        private String label;

        public StaticNested(String label) {
            this.label = label;
        }

        public String getLabel() {
            return label;
        }
    }

    public class Inner {
        public int doubleValue() {
            return value * 2;
        }
    }
}
