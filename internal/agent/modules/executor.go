package modules

import (
    "bytes"
    "os/exec"
    "runtime"
)

type ExecModule struct{}

func (e *ExecModule) Name() string {
    return "exec"
}

func (e *ExecModule) Execute(data []byte) ([]byte, error) {
    command := string(data)
    var cmd *exec.Cmd
    if runtime.GOOS == "windows" {
        cmd = exec.Command("cmd", "/c", command)
    } else {
        cmd = exec.Command("/bin/sh", "-c", command)
    }

    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    err := cmd.Run()
    if err != nil {
        return []byte(stderr.String()), err
    }
    return stdout.Bytes(), nil
}
