package svc;

public class Validator {
    public static boolean isValid(String input) {
        return input != null && !isEmpty(input);
    }

    public static boolean isEmpty(String input) {
        return input.length() == 0;
    }
}
