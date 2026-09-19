//go:build windows && integration

package credential

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"testing"

	"queueb.org/keyring-winbridge/internal/bridge"
)

func TestWindowsCredentialCRUD(t *testing.T) {
	backend := New()
	service := "queueb.org/keyring-winbridge/integration-test"
	account := "test-" + randomHex(t, 8)
	secret := randomBytes(t, 32)

	if err := backend.Probe(t.Context()); err != nil {
		t.Fatalf("Probe() error = %v", err)
	}
	if _, err := backend.Get(t.Context(), service, account); !errors.Is(err, bridge.ErrNotFound) {
		t.Fatalf("initial Get() error = %v, want %v", err, bridge.ErrNotFound)
	}
	t.Cleanup(func() {
		if _, err := backend.Delete(context.Background(), service, account); err != nil {
			t.Errorf("cleanup Delete() error = %v", err)
		}
	})

	if err := backend.Set(t.Context(), service, account, secret); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	got, err := backend.Get(t.Context(), service, account)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !bytes.Equal(got, secret) {
		t.Fatal("Get() returned a different secret")
	}
	deleted, err := backend.Delete(t.Context(), service, account)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() = false, want true")
	}
	if _, err := backend.Get(t.Context(), service, account); !errors.Is(err, bridge.ErrNotFound) {
		t.Fatalf("final Get() error = %v, want %v", err, bridge.ErrNotFound)
	}
}

func randomHex(t *testing.T, size int) string {
	t.Helper()
	return hex.EncodeToString(randomBytes(t, size))
}

func randomBytes(t *testing.T, size int) []byte {
	t.Helper()
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		t.Fatalf("rand.Read() error = %v", err)
	}
	return value
}
