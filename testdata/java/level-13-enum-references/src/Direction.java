package nav;

public enum Direction {
    NORTH, SOUTH, EAST, WEST;

    public Direction opposite() {
        switch (this) {
            case NORTH: return SOUTH;
            case SOUTH: return NORTH;
            case EAST: return WEST;
            default: return EAST;
        }
    }

    public boolean isVertical() {
        return this == NORTH || this == SOUTH;
    }
}
