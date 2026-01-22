package main

import (
	"bytes"
	"log"
	"testing"
)

func TestLogSomething(t *testing.T) {
	var buf bytes.Buffer
	orig := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() {
		log.SetOutput(orig)
	})

	log.Printf("jax-memory log test")
	if buf.Len() == 0 {
		t.Fatalf("expected log output")
	}
}
