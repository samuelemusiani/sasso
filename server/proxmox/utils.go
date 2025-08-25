package proxmox

import (
	"fmt"
	"strings"
)

const base62Alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// EncodeBase62 encodes a uint32 into a base62 string
func EncodeBase62(num uint32) string {
	if num == 0 {
		return string(base62Alphabet[0])
	}
	var sb strings.Builder
	for num > 0 {
		remainder := num % 62
		sb.WriteByte(base62Alphabet[remainder])
		num /= 62
	}
	// reverse since we construct in reverse order
	runes := []rune(sb.String())
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// DecodeBase62 decodes a base62 string into a uint32
func DecodeBase62(s string) (uint32, error) {
	var num uint32
	for _, c := range s {
		index := strings.IndexRune(base62Alphabet, c)
		if index == -1 {
			return 0, fmt.Errorf("invalid character: %c", c)
		}
		num = num*62 + uint32(index)
	}
	return num, nil
}
