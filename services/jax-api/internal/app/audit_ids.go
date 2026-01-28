package app

import (
	"crypto/rand"
	"fmt"
	"time"
)

func NewCorrelationID() string {
	return newAuditID("corr")
}

func newAuditID(prefix string) string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UTC().UnixNano())
	}
	return fmt.Sprintf("%s_%d_%x", prefix, time.Now().UTC().UnixNano(), buf)
}
