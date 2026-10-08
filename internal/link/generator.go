package link

import (
	"crypto/sha256"
	"fmt"
)

const base62Chars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func GenerateCode(longURL string, attempt int) string {
	input := longURL
	if attempt > 0 {
		input = fmt.Sprintf("%s#salt%d", longURL, attempt)
	}

	hash := sha256.Sum256([]byte(input))

	var code []byte
	for i := 0; i < 6; i++ {
		idx := int(hash[i]) % len(base62Chars)
		code = append(code, base62Chars[idx])
	}

	return string(code)
}
