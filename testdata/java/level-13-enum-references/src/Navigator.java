package nav;

public class Navigator {
    private Direction heading;

    public Navigator(Direction heading) {
        this.heading = heading;
    }

    public void turnAround() {
        heading = heading.opposite();
    }

    public String describe() {
        if (heading.isVertical()) {
            return "vertical";
        }
        return "horizontal";
    }
}
