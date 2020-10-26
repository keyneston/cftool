package helpers

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

func HashFile(filename string) (string, error) {
	hasher := sha256.New()
	f, err := os.Open(filename)
	if err != nil {
		return "", fmt.Errorf("error hashing %q: %v", filename, err)
	}
	defer f.Close()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", fmt.Errorf("error hashing %q: %v", filename, err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func HashString(input string) string {
	hasher := sha256.New()
	hasher.Write([]byte(input))

	return hex.EncodeToString(hasher.Sum(nil))
}
