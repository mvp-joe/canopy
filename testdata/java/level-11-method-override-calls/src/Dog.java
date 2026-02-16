package zoo;

public class Dog implements Animal {
    public String speak() {
        return "Woof";
    }

    public String name() {
        return "Dog";
    }

    public String fetch() {
        return speak() + " fetch!";
    }
}
