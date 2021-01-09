package server

import (
	"strings"
	"testing"

	"github.com/awnumar/memguard"
)

func TestServer(t *testing.T) {
	memguard.CatchInterrupt()
	defer memguard.Purge()
	r := strings.NewReader("foo")
	buffer, err := memguard.NewBufferFromEntireReader(r)
	if err != nil {
		t.Fatalf("error creating locked buffer: %v", err)
	}
	if buffer == nil {
		t.Fatal("buffer is nil")
	}
}
