package main

import (
    "nedo-phantom/internal/c2/core"
)

func main() {
    srv := core.NewServer(":4444")
    srv.Run()
}
