package crypto

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "crypto/rsa"
    "crypto/sha256"
    "crypto/x509"
    "encoding/pem"
    "fmt"
)

type ServerCrypto struct {
    symmetricKey []byte
}

func NewServerCryptoFromKey(key []byte) *ServerCrypto {
    return &ServerCrypto{
        symmetricKey: key,
    }
}

// Decrypt расшифровывает данные через AES-GCM
func (c *ServerCrypto) Decrypt(data []byte) ([]byte, error) {
    if c.symmetricKey == nil {
        return nil, fmt.Errorf("symmetric key not set")
    }

    block, err := aes.NewCipher(c.symmetricKey)
    if err != nil {
        return nil, fmt.Errorf("failed to create AES cipher: %w", err)
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, fmt.Errorf("failed to create GCM: %w", err)
    }

    nonceSize := gcm.NonceSize()
    if len(data) < nonceSize {
        return nil, fmt.Errorf("ciphertext too short")
    }

    nonce := data[:nonceSize]
    ciphertext := data[nonceSize:]

    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to decrypt: %w", err)
    }

    return plaintext, nil
}

// Encrypt шифрует данные через AES-GCM
func (c *ServerCrypto) Encrypt(data []byte) ([]byte, error) {
    if c.symmetricKey == nil {
        return nil, fmt.Errorf("symmetric key not set")
    }

    block, err := aes.NewCipher(c.symmetricKey)
    if err != nil {
        return nil, fmt.Errorf("failed to create AES cipher: %w", err)
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, fmt.Errorf("failed to create GCM: %w", err)
    }

    nonce := make([]byte, gcm.NonceSize())
    if _, err := rand.Read(nonce); err != nil {
        return nil, fmt.Errorf("failed to generate nonce: %w", err)
    }

    ciphertext := gcm.Seal(nonce, nonce, data, nil)
    return ciphertext, nil
}

// GenerateSymmetricKey генерирует новый AES-256 ключ
func GenerateSymmetricKey() ([]byte, error) {
    key := make([]byte, 32)
    _, err := rand.Read(key)
    if err != nil {
        return nil, fmt.Errorf("failed to generate symmetric key: %w", err)
    }
    return key, nil
}

// EncryptSymmetricKey шифрует AES ключ через RSA публичный ключ агента
func EncryptSymmetricKey(publicKeyPEM string, symmetricKey []byte) ([]byte, error) {
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
