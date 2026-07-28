package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"crypto/x509"

	"encoding/pem"
	"fmt"
)

const (
    AESKeySize = 32 // 256 бит
    RSAKeySize = 2048
)

type Crypto struct {
	privateKey *rsa.PrivateKey
	publicKey *rsa.PublicKey

	symmetricKey []byte
}

func NewCrypto() (*Crypto, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, RSAKeySize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key pair: %w", err)
	}

	return &Crypto{
		privateKey: privateKey,
		publicKey: &privateKey.PublicKey,
	}, nil
}

func (c *Crypto) IsReady() bool {
    return c.symmetricKey != nil
}

func (c *Crypto) GetPublicKeyPEM() (string, error) {
  // Экспортировать публичный ключ в PEM формат
	publicPEM, err := x509.MarshalPKIXPublicKey(c.publicKey)
	if err != nil {
		return "", fmt.Errorf("failed to export public key: %w", err)
	}

	pemBlock := &pem.Block{
		Type: "PUBLIC KEY",
		Bytes: publicPEM,
	}
	
	pemBytes := pem.EncodeToMemory(pemBlock)

	return string(pemBytes), nil
}

func (c *Crypto) SetSymmetricKey(encryptedKey []byte) error {
  // https://pkg.go.dev/crypto/rsa#DecryptOAEP
	hash := sha256.New()
	symmetricKey, err := rsa.DecryptOAEP(hash, nil, c.privateKey, encryptedKey, nil)
	if err != nil {
		return fmt.Errorf("failed to decrypt key: %w", err)
	}

	if len(symmetricKey) != AESKeySize {
		return fmt.Errorf("invalid symmetric key length: expected 32 bytes, got %d", len(symmetricKey))
  }
	
	c.symmetricKey = symmetricKey
	return nil
}

func (c *Crypto) Encrypt(data []byte) ([]byte, error) {
  // AES-GCM шифрование данных
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

func (c *Crypto) Decrypt(data []byte) ([]byte, error) {
  // AES-GCM расшифровка данных
	if c.symmetricKey == nil {
		return nil, fmt.Errorf("symmetric key not set")
	}

	block, err := aes.NewCipher(c.symmetricKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM %w", err)
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
