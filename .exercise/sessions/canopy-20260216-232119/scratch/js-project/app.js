const { EventEmitter } = require('events');
const { formatDate, parseConfig } = require('./utils');

class TaskManager extends EventEmitter {
  constructor() {
    super();
    this.tasks = [];
    this.nextId = 1;
  }

  addTask(title, priority = 'medium') {
    const task = {
      id: this.nextId++,
      title,
      priority,
      completed: false,
      createdAt: formatDate(new Date()),
    };
    this.tasks.push(task);
    this.emit('taskAdded', task);
    return task;
  }

  completeTask(id) {
    const task = this.tasks.find(t => t.id === id);
    if (!task) throw new Error(`Task ${id} not found`);
    task.completed = true;
    this.emit('taskCompleted', task);
    return task;
  }

  getActiveTasks() {
    return this.tasks.filter(t => !t.completed);
  }

  getTasksByPriority(priority) {
    return this.tasks.filter(t => t.priority === priority);
  }
}

function createManager(configPath) {
  const config = parseConfig(configPath);
  const manager = new TaskManager();
  if (config.maxTasks) {
    manager.maxTasks = config.maxTasks;
  }
  return manager;
}

module.exports = { TaskManager, createManager };
