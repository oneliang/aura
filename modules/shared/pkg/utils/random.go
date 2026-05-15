// Package utils provides common utility functions.
package utils

import (
	"crypto/rand"
	"math/big"
)

const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// RandString generates a cryptographically secure random alphanumeric string of length n.
func RandString(n int) (string, error) {
	if n <= 0 {
		return "", nil
	}

	b := make([]byte, n)
	lettersLen := big.NewInt(int64(len(letters)))

	for i := range b {
		randomIdx, err := rand.Int(rand.Reader, lettersLen)
		if err != nil {
			return "", err
		}
		b[i] = letters[randomIdx.Int64()]
	}

	return string(b), nil
}

// MustRandString is like RandString but panics on error.
// Use only in tests or initialization code where randomness is critical.
func MustRandString(n int) string {
	s, err := RandString(n)
	if err != nil {
		panic(err)
	}
	return s
}
