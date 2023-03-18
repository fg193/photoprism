package rnd

import (
	"encoding/binary"
	"time"
)

const (
	PrefixNone  = byte(0)
	PrefixMixed = byte('*')
	Epoch       = 1577836800 // 2020-01-01 UTC
)

// GenerateUID returns a unique id with prefix as string.
func GenerateUID(prefix byte) string {
	result := make([]byte, 9)
	result[0] = prefix
	now := uint32(time.Now().UTC().Unix() - Epoch)
	binary.BigEndian.PutUint32(result[4:], now)
	crockford.Encode(result[1:], result[4:])
	return string(result[:4]) + GenerateToken(5)
}
