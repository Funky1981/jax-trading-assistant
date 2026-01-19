package observability

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

func NewRunID() string {
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("run_%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("run_%d_%s", time.Now().UnixNano(), hex.EncodeToString(buf))
}
