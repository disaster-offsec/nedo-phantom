package db

import (
    "sync"
    "nedo-phantom/internal/common"
)

type TaskQueue struct {
    mu      sync.Mutex
    tasks   []common.Task
    pending map[string]common.Task
}

func NewTaskQueue() *TaskQueue {
    return &TaskQueue{
        tasks:   make([]common.Task, 0),
        pending: make(map[string]common.Task),
    }
}

func (q *TaskQueue) AddTask(task common.Task) {
    q.mu.Lock()
    defer q.mu.Unlock()
    q.tasks = append(q.tasks, task)
    q.pending[task.ID] = task
}

func (q *TaskQueue) GetNextTask() *common.Task {
    q.mu.Lock()
    defer q.mu.Unlock()

    for i, task := range q.tasks {
        if task.Status == "pending" {
            task.Status = "running"
            q.tasks[i] = task
            return &task
        }
    }
    return nil
}

func (q *TaskQueue) GetTasks() []common.Task {
    q.mu.Lock()
    defer q.mu.Unlock()
    return q.tasks
}
