package main

import (
    "nedo-phantom/internal/c2/server"
)

func main() {
    srv := server.NewServer(":4444")
    srv.Run()
}
