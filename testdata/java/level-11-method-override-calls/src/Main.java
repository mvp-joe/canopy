package zoo;

public class Main {
    public static void describe(Animal a) {
        String sound = a.speak();
        String id = a.name();
        System.out.println(id + " says " + sound);
    }

    public static void main(String[] args) {
        Dog d = new Dog();
        describe(d);
        String result = d.fetch();
        System.out.println(result);
    }
}
