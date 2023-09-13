package encryptor

import (
	"encoding/base64"

	"github.com/gtank/cryptopasta"
	"github.com/pkg/errors"
)

type Encryptor struct {
	secret *[32]byte
}

func NewEncryptor(secretString string) *Encryptor {
	secret := &[32]byte{}
	copy(secret[:], secretString)
	return &Encryptor{secret: secret}
}

func (e *Encryptor) EncryptString(plaintext string) (string, error) {
	encryptedBytes, err := cryptopasta.Encrypt([]byte(plaintext), e.secret)
	if err != nil {
		return "", errors.Wrap(err, "failed to encrypt string")
	}
	b64 := base64.StdEncoding.EncodeToString(encryptedBytes)
	return b64, nil
}

func (e *Encryptor) DecryptString(ciphertext string) (string, error) {
	decodedBytes, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", errors.Wrap(err, "failed to decode string")
	}
	decryptedBytes, err := cryptopasta.Decrypt(decodedBytes, e.secret)
	if err != nil {
		return "", errors.Wrap(err, "failed to decrypt string")
	}
	return string(decryptedBytes), nil
}
