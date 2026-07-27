package common

type Task struct {
    ID     string      `json:"id"`
    Type   string      `json:"type"`   // exec, file, sleep, etc.
    Data   interface{} `json:"data"`
    Status string      `json:"status"` // pending, running, done, error
}

type TaskResult struct {
    TaskID string `json:"task_id"`
    Output string `json:"output"`
    Error  string `json:"error,omitempty"`
}
