// This store is used to store current tasks that are being processed. It
// stores a list of tasks and has a function to add a task to the list and
// a function to remove a task when it is completed.

import { writable, get } from "svelte/store";

interface Task {
  id: string;
  message: string;
}
const taskList = writable<Task[]>([]);
const showTasks = writable<boolean>(false);

function startTask(task: Task) {
  // Add the new task as the first item in the list
  taskList.update((tasks) => [task, ...tasks]);
  showTasks.set(true);
}

function endTask(id: string) {
  // Remove the task from the list
  taskList.update((tasks) => tasks.filter((t) => t.id !== id));
  if (get(taskList).length === 0) {
    showTasks.set(false);
  }
}

function firstMessage() {
  if (get(showTasks)) {
    return get(taskList)[0]?.message;
  }
  return "";
}

export const tasks = {
  subscribe: taskList.subscribe,
  showTasks: showTasks.subscribe,
  startTask,
  endTask,
  firstMessage,
};
