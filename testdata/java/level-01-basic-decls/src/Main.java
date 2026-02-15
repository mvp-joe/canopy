public class Main {
    public static final int MAX_RETRIES = 3;

    public static boolean debug = false;

    public static String hello() {
        return "hello";
    }

    public static int add(int a, int b) {
        return a + b;
    }

    public static void main(String[] args) {
        System.out.println(hello());
        int result = add(1, 2);
    }
}
