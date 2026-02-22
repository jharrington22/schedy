package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

type AEAD struct{ aead cipher.AEAD }

func New(key []byte) (*AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil { return nil, err }
	a, err := cipher.NewGCM(block)
	if err != nil { return nil, err }
	return &AEAD{aead: a}, nil
}

func (a *AEAD) EncryptToString(plaintext string) (string, error) {
	nonce := make([]byte, a.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil { return "", err }
	ct := a.aead.Seal(nil, nonce, []byte(plaintext), nil)
	buf := append(nonce, ct...)
	return base64.RawStdEncoding.EncodeToString(buf), nil
}

func (a *AEAD) DecryptString(ciphertextB64 string) (string, error) {
	buf, err := base64.RawStdEncoding.DecodeString(ciphertextB64)
	if err != nil { return "", err }
	ns := a.aead.NonceSize()
	if len(buf) < ns { return "", fmt.Errorf("ciphertext too short") }
	nonce := buf[:ns]
	ct := buf[ns:]
	pt, err := a.aead.Open(nil, nonce, ct, nil)
	if err != nil { return "", err }
	return string(pt), nil
}
