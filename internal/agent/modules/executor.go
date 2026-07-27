package modules

import (
    "bytes"
    "os/exec"
    "runtime"
)

type Executor struct{}

func (e *Executor) Execute(command string) (string, error) {
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
        return stderr.String(), err
    }
    return stdout.String(), nil
}
