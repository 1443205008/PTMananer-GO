// Package cryptoutil: scrypt 密码哈希（与 TS 版 auth/password.ts 格式兼容）
// 与 AES-256-GCM 凭据加解密（与 TS 版 crypto/credential-crypto.service.ts 兼容）。
package cryptoutil

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/scrypt"
)

// ─── 密码（scrypt，格式 scrypt$<salt hex>$<hash hex>，可被 TS 版验证）──────

const (
	saltBytes = 16
	keyLength = 64
)

func HashPassword(password string) (string, error) {
	salt := make([]byte, saltBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash, err := scrypt.Key([]byte(password), salt, 16384, 8, 1, keyLength)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("scrypt$%s$%s", hex.EncodeToString(salt), hex.EncodeToString(hash)), nil
}

// VerifyPassword 校验密码，格式异常时返回 false（对齐 TS 版行为）。
func VerifyPassword(password, encodedHash string) bool {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 3 || parts[0] != "scrypt" {
		return false
	}
	salt, err1 := hex.DecodeString(parts[1])
	expected, err2 := hex.DecodeString(parts[2])
	if err1 != nil || err2 != nil {
		return false
	}
	// 长度必须固定，不能取自库里存的哈希长度（对齐 TS 版的安全约束）
	if len(expected) != keyLength || len(salt) != saltBytes {
		return false
	}
	actual, err := scrypt.Key([]byte(password), salt, 16384, 8, 1, keyLength)
	if err != nil {
		return false
	}
	return constantTimeEqual(expected, actual)
}

func constantTimeEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := range a {
		v |= a[i] ^ b[i]
	}
	return v == 0
}

// ─── AES-256-GCM 凭据加解密 ────────────────────────────────────────────────

const (
	gcmIVLength  = 12 // 96 bits
	gcmTagLength = 16 // 128 bits
)

type CredentialCrypto struct {
	key []byte
}

// NewCredentialCrypto 接受 64 位 hex 字符串（32 字节 key）。
func NewCredentialCrypto(hexKey string) (*CredentialCrypto, error) {
	if len(hexKey) != 64 {
		return nil, errors.New("CREDENTIAL_ENCRYPTION_KEY must be a 64-character hex string (32 bytes). Generate one with: openssl rand -hex 32")
	}
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("invalid CREDENTIAL_ENCRYPTION_KEY: %w", err)
	}
	return &CredentialCrypto{key: key}, nil
}

type EncryptedCredential struct {
	EncryptedAPIKey string // hex
	IV              string // hex
	AuthTag         string // hex
}

func (c *CredentialCrypto) Encrypt(plaintext string) (EncryptedCredential, error) {
	iv := make([]byte, gcmIVLength)
	if _, err := rand.Read(iv); err != nil {
		return EncryptedCredential{}, err
	}
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return EncryptedCredential{}, err
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, gcmIVLength)
	if err != nil {
		return EncryptedCredential{}, err
	}
	// Node crypto 产出的 tag 在密文尾部；gcm.Seal 同样把 tag 追加在尾部，二者兼容。
	ct := gcm.Seal(nil, iv, []byte(plaintext), nil)
	ciphertext := ct[:len(ct)-gcmTagLength]
	tag := ct[len(ct)-gcmTagLength:]
	return EncryptedCredential{
		EncryptedAPIKey: hex.EncodeToString(ciphertext),
		IV:              hex.EncodeToString(iv),
		AuthTag:         hex.EncodeToString(tag),
	}, nil
}

func (c *CredentialCrypto) Decrypt(encryptedAPIKey, ivHex, authTagHex string) (string, error) {
	iv, err := hex.DecodeString(ivHex)
	if err != nil {
		return "", err
	}
	tag, err := hex.DecodeString(authTagHex)
	if err != nil {
		return "", err
	}
	ct, err := hex.DecodeString(encryptedAPIKey)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, gcmIVLength)
	if err != nil {
		return "", err
	}
	// Node decipheriv 的密文不含 tag；gcm.Open 期望 ciphertext||tag
	combined := append(append([]byte{}, ct...), tag...)
	plaintext, err := gcm.Open(nil, iv, combined, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}
	return string(plaintext), nil
}
