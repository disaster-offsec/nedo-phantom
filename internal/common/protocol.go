package common

import "encoding/json"

type Message struct {
    Type   string      `json:"type"`
    Task   *Task       `json:"task,omitempty"`
    Result *TaskResult `json:"result,omitempty"`
}

type Task struct {
    ID     string      `json:"id"`
    Type   string      `json:"type"`
    Data   interface{} `json:"data"`
    Status string      `json:"status"`
}

type TaskResult struct {
    TaskID string `json:"task_id"`
    Output string `json:"output"`
    Error  string `json:"error,omitempty"`
}

// ParseMessage парсит JSON в Message
func ParseMessage(data []byte) (*Message, error) {
    var msg Message
    if err := json.Unmarshal(data, &msg); err != nil {
        return nil, err
    }
    return &msg, nil
}

// MarshalMessage сериализует Message в JSON
func MarshalMessage(msg *Message) ([]byte, error) {
    return json.Marshal(msg)
}
