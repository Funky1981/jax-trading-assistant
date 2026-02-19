package observability

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// NewRunID generates a unique identifier for an orchestration run.
func NewRunID() string {
	return newID("run")
}

// NewFlowID generates a unique identifier for a full trade-decision flow
// (quote ingested → signal → orchestration → approval → execution).
func NewFlowID() string {
	return newID("flow")
}

func newID(prefix string) string {
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return fmt.Sprintf("%s_%d_%s", prefix, time.Now().UnixNano(), hex.EncodeToString(buf))
}
