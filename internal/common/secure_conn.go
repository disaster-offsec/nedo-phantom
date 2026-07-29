package common

import (
    "encoding/binary"
    "fmt"
    "io"
    "net"
    "time"
)

// SecureConn оборачивает net.Conn и автоматически шифрует/расшифровывает данные
type SecureConn struct {
    conn   net.Conn
    cipher Cipher // интерфейс
}

// NewSecureConn создает новое зашифрованное соединение
func NewSecureConn(conn net.Conn, cipher Cipher) *SecureConn {
    return &SecureConn{
        conn:   conn,
        cipher: cipher,
    }
}

// Write шифрует и записывает данные
func (s *SecureConn) Write(data []byte) (int, error) {
    encrypted, err := s.cipher.Encrypt(data)
    if err != nil {
        return 0, fmt.Errorf("encrypt failed: %w", err)
    }

    length := uint32(len(encrypted))
    lengthBuf := make([]byte, 4)
    binary.BigEndian.PutUint32(lengthBuf, length)

    if _, err := s.conn.Write(lengthBuf); err != nil {
        return 0, fmt.Errorf("write length failed: %w", err)
    }

    if _, err := s.conn.Write(encrypted); err != nil {
        return 0, fmt.Errorf("write encrypted data failed: %w", err)
    }

    return len(data), nil
}

// Read читает и расшифровывает данные
func (s *SecureConn) Read(buf []byte) (int, error) {
    lengthBuf := make([]byte, 4)
    if _, err := io.ReadFull(s.conn, lengthBuf); err != nil {
        return 0, fmt.Errorf("read length failed: %w", err)
    }

    length := binary.BigEndian.Uint32(lengthBuf)

    encrypted := make([]byte, length)
    if _, err := io.ReadFull(s.conn, encrypted); err != nil {
        return 0, fmt.Errorf("read encrypted data failed: %w", err)
    }

    plaintext, err := s.cipher.Decrypt(encrypted)
    if err != nil {
        return 0, fmt.Errorf("decrypt failed: %w", err)
    }

    n := copy(buf, plaintext)
    if n < len(plaintext) {
        return n, fmt.Errorf("buffer too small: need %d bytes, got %d", len(plaintext), len(buf))
    }

    return n, nil
}

// Close закрывает соединение
func (s *SecureConn) Close() error {
    return s.conn.Close()
}

// LocalAddr возвращает локальный адрес
func (s *SecureConn) LocalAddr() net.Addr {
    return s.conn.LocalAddr()
}

// RemoteAddr возвращает удаленный адрес
func (s *SecureConn) RemoteAddr() net.Addr {
    return s.conn.RemoteAddr()
}

// SetDeadline устанавливает дедлайн
func (s *SecureConn) SetDeadline(t time.Time) error {
    return s.conn.SetDeadline(t)
}

// SetReadDeadline устанавливает дедлайн на чтение
func (s *SecureConn) SetReadDeadline(t time.Time) error {
    return s.conn.SetReadDeadline(t)
}

// SetWriteDeadline устанавливает дедлайн на запись
func (s *SecureConn) SetWriteDeadline(t time.Time) error {
    return s.conn.SetWriteDeadline(t)
}
