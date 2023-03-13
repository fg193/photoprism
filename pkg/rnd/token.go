package rnd

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
)

var (
	crockford = base32.NewEncoding(CharsetBase32)
)

// GenerateToken returns a random token with length of up to 10 characters.
func GenerateToken(size uint) string {
	if size > 10 || size < 1 {
		panic(fmt.Sprintf("size out of range: %d", size))
	}

	b := make([]byte, 7)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}

	return crockford.EncodeToString(b)[:size]
}
