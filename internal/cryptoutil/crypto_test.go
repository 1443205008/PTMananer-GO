package cryptoutil

import (
	"testing"
)

func TestPasswordHashVerify(t *testing.T) {
	h, err := HashPassword("s3cret-pw")
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyPassword("s3cret-pw", h) {
		t.Fatal("correct password should verify")
	}
	if VerifyPassword("wrong", h) {
		t.Fatal("wrong password should fail")
	}
	// 格式异常 → false
	if VerifyPassword("x", "garbage") || VerifyPassword("x", "scrypt$zz$zz") || VerifyPassword("x", "") {
		t.Fatal("malformed hash should fail")
	}
}

func TestAESGCMRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	hexKey := hexEncode(key)
	c, err := NewCredentialCrypto(hexKey)
	if err != nil {
		t.Fatal(err)
	}

	cred, err := c.Encrypt("mteam-api-key-abc123")
	if err != nil {
		t.Fatal(err)
	}
	if cred.EncryptedAPIKey == "" || len(cred.IV) != 24 || len(cred.AuthTag) != 32 {
		t.Fatalf("bad cred: %+v", cred)
	}
	plain, err := c.Decrypt(cred.EncryptedAPIKey, cred.IV, cred.AuthTag)
	if err != nil {
		t.Fatal(err)
	}
	if plain != "mteam-api-key-abc123" {
		t.Fatalf("round trip mismatch: %q", plain)
	}

	// 相同明文两次加密，IV 不同（随机 IV）
	cred2, _ := c.Encrypt("mteam-api-key-abc123")
	if cred.IV == cred2.IV {
		t.Fatal("IV should be random per encryption")
	}

	// 篡改密文 → 解密失败
	if _, err := c.Decrypt("00"+cred.EncryptedAPIKey[2:], cred.IV, cred.AuthTag); err == nil {
		t.Fatal("tampered ciphertext should fail")
	}
}

func TestKeyValidation(t *testing.T) {
	if _, err := NewCredentialCrypto("tooshort"); err == nil {
		t.Fatal("short key should be rejected")
	}
	if _, err := NewCredentialCrypto("zz" + string(make([]byte, 62))); err == nil {
		t.Fatal("non-hex key should be rejected")
	}
}

func hexEncode(b []byte) string {
	const hexDigits = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, v := range b {
		out[i*2] = hexDigits[v>>4]
		out[i*2+1] = hexDigits[v&0x0f]
	}
	return string(out)
}
