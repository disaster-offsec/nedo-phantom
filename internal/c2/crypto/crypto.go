package crypto

import (
    "crypto/rand"
    "crypto/rsa"
    "crypto/sha256"
    "crypto/x509"
    "encoding/pem"
    "fmt"
)

type ServerCrypto struct {
    // У сервера нет своей RSA пары (он не расшифровывает)
    // Вместо этого он хранит публичные ключи агентов
    // и генерирует AES ключи для каждого агента
}

// GenerateSymmetricKey генерирует новый AES-256 ключ
func GenerateSymmetricKey() ([]byte, error) {
    key := make([]byte, 32) // 256 бит
    _, err := rand.Read(key)
    if err != nil {
        return nil, fmt.Errorf("failed to generate symmetric key: %w", err)
    }
    return key, nil
}

// EncryptSymmetricKey шифрует AES ключ через RSA публичный ключ агента
func EncryptSymmetricKey(publicKeyPEM string, symmetricKey []byte) ([]byte, error) {
    // 1. Парсим PEM в публичный ключ
    block, _ := pem.Decode([]byte(publicKeyPEM))
    if block == nil {
        return nil, fmt.Errorf("failed to decode PEM")
    }
    
    publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
    if err != nil {
        return nil, fmt.Errorf("failed to parse public key: %w", err)
    }
    
    rsaPublicKey, ok := publicKey.(*rsa.PublicKey)
    if !ok {
        return nil, fmt.Errorf("not an RSA public key")
    }
    
    // 2. Шифруем AES ключ через RSA
    encryptedKey, err := rsa.EncryptOAEP(
        sha256.New(),
        rand.Reader,
        rsaPublicKey,
        symmetricKey,
        nil,
    )
    if err != nil {
        return nil, fmt.Errorf("failed to encrypt symmetric key: %w", err)
    }
    
    return encryptedKey, nil
}
