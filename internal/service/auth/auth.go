package auth

import (
	"crypto/rand"
	"encoding/base64"

	"github.com/pkg/errors"
)

type Storage interface {
	SavePassword(userID int64, password string) error
	PopPassword(userID int64, verifyFunc func(string) error) error
}

type Service struct {
	store Storage
}

func NewService(store Storage) *Service {
	return &Service{
		store: store,
	}
}

func (s *Service) GeneratePassword(userID int64) (string, error) {
	password, err := s.generatePassword()
	if err != nil {
		return "", errors.Wrap(err, "generating password")
	}
	if err := s.store.SavePassword(userID, password); err != nil {
		return "", err
	}
	return password, nil
}

func (s *Service) VerifyPassword(userID int64, password string) error {
	return s.store.PopPassword(userID, func(storedPassword string) error {
		if password != storedPassword {
			return errors.New("invalid password")
		}
		return nil
	})
}

// generatePassword generates a random string and signs it with the private key.
func (s *Service) generatePassword() (password string, err error) {
	// Generate a random string of length 32
	randomBytes := make([]byte, 32)
	_, err = rand.Read(randomBytes)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(randomBytes), nil
}
