package core

import (
    "net"
    "time"
    "nedo-phantom/internal/common"
)

// AgentSession представляет сессию подключенного агента
type AgentSession struct {
    conn       net.Conn
    hostname   string
    lastSeen   time.Time
    publicKey  string
    aesKey     []byte
    secureConn *common.SecureConn
}

// NewAgentSession создает новую сессию агента
func NewAgentSession(conn net.Conn, hostname string) *AgentSession {
    return &AgentSession{
        conn:      conn,
        hostname:  hostname,
        lastSeen:  time.Now(),
    }
}

// UpdateLastSeen обновляет время последнего контакта с агентом
func (s *AgentSession) UpdateLastSeen() {
    s.lastSeen = time.Now()
}

// SetKeys устанавливает ключи агента (публичный RSA и AES)
func (s *AgentSession) SetKeys(publicKey string, aesKey []byte) {
    s.publicKey = publicKey
    s.aesKey = aesKey
}

// SetSecureConn устанавливает зашифрованное соединение
func (s *AgentSession) SetSecureConn(secureConn *common.SecureConn) {
    s.secureConn = secureConn
}

// GetHostname возвращает имя хоста агента
func (s *AgentSession) GetHostname() string {
    return s.hostname
}

// GetLastSeen возвращает время последнего контакта
func (s *AgentSession) GetLastSeen() time.Time {
    return s.lastSeen
}

// IsOnline проверяет, онлайн ли агент (последний контакт был менее 30 секунд назад)
func (s *AgentSession) IsOnline() bool {
    return time.Since(s.lastSeen) < 30*time.Second
}

// Close закрывает соединение агента
func (s *AgentSession) Close() error {
    if s.conn != nil {
        return s.conn.Close()
    }
    return nil
}
