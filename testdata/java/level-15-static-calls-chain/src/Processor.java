package svc;

public class Processor {
    private String data;

    public Processor(String data) {
        this.data = data;
    }

    public String process() {
        if (Validator.isValid(data)) {
            return transform(data);
        }
        return "";
    }

    private String transform(String input) {
        return input.toUpperCase();
    }
}
