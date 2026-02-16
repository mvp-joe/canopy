public class Task {
    private String name;
    private Priority priority;

    public Task(String name, Priority priority) {
        this.name = name;
        this.priority = priority;
    }

    public Priority getPriority() {
        return priority;
    }

    public boolean isUrgent() {
        return priority == Priority.CRITICAL;
    }
}
