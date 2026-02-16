public class Box<T> {
    private T value;

    public Box(T value) {
        this.value = value;
    }

    public T getValue() {
        return value;
    }

    public <U extends Comparable<U>> int compareTo(U other) {
        return 0;
    }

    public static <V> Box<V> of(V item) {
        return new Box<>(item);
    }
}
