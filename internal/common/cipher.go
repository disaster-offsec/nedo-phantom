package common

// Cipher — интерфейс для шифрования/расшифровки данных
type Cipher interface {
    Encrypt(data []byte) ([]byte, error)
    Decrypt(data []byte) ([]byte, error)
}
