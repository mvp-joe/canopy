public enum Color {
    RED("#FF0000"),
    GREEN("#00FF00"),
    BLUE("#0000FF");

    private final String hex;

    Color(String hex) {
        this.hex = hex;
    }

    public String getHex() {
        return hex;
    }

    public static Color fromHex(String hex) {
        for (Color c : values()) {
            if (c.hex.equals(hex)) {
                return c;
            }
        }
        return null;
    }
}
