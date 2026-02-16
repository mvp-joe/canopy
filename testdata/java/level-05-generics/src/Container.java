public class Container<T> {
    private T value;

    public Container(T value) {
        this.value = value;
    }

    public T getValue() {
        return value;
    }

    public <R> R transform(Transformer<T, R> fn) {
        return fn.apply(value);
    }
}
