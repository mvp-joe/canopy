import java.util.List;
import java.util.ArrayList;

public class App {
    public static void main(String[] args) {
        TodoList todos = new TodoList();
        todos.add("Buy groceries", Priority.HIGH);
        todos.add("Clean house", Priority.MEDIUM);
        todos.add("Read book", Priority.LOW);

        System.out.println("All todos:");
        for (Todo t : todos.getAll()) {
            System.out.println("  " + t);
        }

        todos.complete(1);
        System.out.println("Active: " + todos.getActive().size());
    }
}

enum Priority {
    LOW, MEDIUM, HIGH
}

class Todo {
    int id;
    String title;
    Priority priority;
    boolean completed;

    Todo(int id, String title, Priority priority) {
        this.id = id;
        this.title = title;
        this.priority = priority;
        this.completed = false;
    }

    public String toString() {
        return String.format("[%s] %s (%s)", completed ? "x" : " ", title, priority);
    }
}

class TodoList {
    private List<Todo> todos = new ArrayList<>();
    private int nextId = 1;

    public Todo add(String title, Priority priority) {
        Todo t = new Todo(nextId++, title, priority);
        todos.add(t);
        return t;
    }

    public void complete(int id) {
        for (Todo t : todos) {
            if (t.id == id) {
                t.completed = true;
                return;
            }
        }
        throw new RuntimeException("Todo not found: " + id);
    }

    public List<Todo> getAll() {
        return new ArrayList<>(todos);
    }

    public List<Todo> getActive() {
        List<Todo> active = new ArrayList<>();
        for (Todo t : todos) {
            if (!t.completed) active.add(t);
        }
        return active;
    }

    public List<Todo> getByPriority(Priority p) {
        List<Todo> result = new ArrayList<>();
        for (Todo t : todos) {
            if (t.priority == p) result.add(t);
        }
        return result;
    }
}
