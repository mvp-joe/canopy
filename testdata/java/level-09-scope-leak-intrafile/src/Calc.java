public class Calc {
    public int compute() {
        return 42;
    }

    public class Inner {
        public int compute() {
            return 99;
        }

        public int run() {
            return compute();
        }
    }
}
