package svc;

public class Runner {
    public static void run() {
        Processor p = new Processor("hello");
        String result = p.process();
        System.out.println(result);
    }
}
