public interface Transformer<A, B> {
    B apply(A input);
}
