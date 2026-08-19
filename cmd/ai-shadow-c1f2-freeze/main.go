package main

import (
	"fmt"
	"os"
	"path/filepath"

	"jax-trading-assistant/internal/modules/aishadow"
)

func main() {
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	path, hash, err := aishadow.GenerateC1F2OfflineFreeze(root, filepath.Join(root, ".runtime", "diagnostics"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("path=%s\nsha256=%s\nprovider_contact=false\ninference=false\n", path, hash)
}
