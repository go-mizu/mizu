package jina

import (
	"fmt"
	"testing"
)

func TestRegister(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode — needs browser + network")
	}

	r := &registrar{}
	key, err := r.Register("", "", true)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	fmt.Printf("KEY: %s\n", key)
	if !jinaKeyRe.MatchString(key) {
		t.Fatalf("bad key format: %s", key)
	}
}
