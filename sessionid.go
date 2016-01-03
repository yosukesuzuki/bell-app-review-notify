package main

import (
	"crypto/rand"
	"encoding/base32"
	"io"
	"strings"
)

func sessionID() string {
	b := make([]byte, 32)
	_, _ = io.ReadFull(rand.Reader, b)
	return strings.TrimRight(base32.StdEncoding.EncodeToString(b), "=")
}
