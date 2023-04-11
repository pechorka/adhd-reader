package filechecksum

import (
	"bytes"
	"crypto/sha256"
	"io"
)

func Calculate(data []byte) []byte {
	hasher := sha256.New()
	// both hasher and bytes.NewReader never return errors
	io.Copy(hasher, bytes.NewReader(data))
	return hasher.Sum(nil)
}
