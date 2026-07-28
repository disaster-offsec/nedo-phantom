package common

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"
	"nedo-phantom/internal/agent/crypto"
)

// SecureConn оборачивает net.Conn и автоматически шифрует/расшифровывает данные
type SecureConn struct {
    conn   net.Conn
    crypto *crypto.Crypto
}

// NewSecureConn создает новое зашифрованное соединение
func NewSecureConn(conn net.Conn, crypto *crypto.Crypto) *SecureConn {
    return &SecureConn{
        conn:   conn,
        crypto: crypto,
    }
}

// Write шифрует и записывает данные
func (s *SecureConn) Write(data []byte) (int, error) {
	// 1. Шифруем данные через crypto.Encrypt()
	encrypted, err := s.crypto.Encrypt(data)
	if err != nil {
		return 0, fmt.Errorf("encrypt failed: %w", err)
	}

	// 2. Отправляем длину зашифрованных данных (4 байта)
	length := uint32(len(encrypted))
	lengthBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthBuf, length)

	if _, err := s.conn.Write(lengthBuf); err != nil {
		return 0, fmt.Errorf("write length failed: %w", err)
	}
	
	// 3. Отправляем зашифрованные данные
	if _, err := s.conn.Write(encrypted); err != nil {
		return 0, fmt.Errorf("write encrypted data failed: %w", err)
	}

	// 4. Возвращаем количество оригинальных байт
	return len(data), nil
}

// Read читает и расшифровывает данные
func (s *SecureConn) Read(buf []byte) (int, error) {
	// 1. Читаем длину зашифрованных данных (4 байта)
	lengthBuf := make([]byte, 4)
	if _, err := io.ReadFull(s.conn, lengthBuf); err != nil {  // <-- io.ReadFull вместо conn.Read
			return 0, fmt.Errorf("read length failed: %w", err)
	}
	
	length := binary.BigEndian.Uint32(lengthBuf)

	// 2. Читаем зашифрованные данные (гарантированно все байты)
	encrypted := make([]byte, length)
	if _, err := io.ReadFull(s.conn, encrypted); err != nil {  // <-- io.ReadFull
			return 0, fmt.Errorf("read encrypted data failed: %w", err)
	}

	// 3. Расшифровываем
	plaintext, err := s.crypto.Decrypt(encrypted)
	if err != nil {
			return 0, fmt.Errorf("decrypt failed: %w", err)
	}

	// 4. Копируем в buf
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
