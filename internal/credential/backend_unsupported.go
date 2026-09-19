//go:build !windows

package credential

import (
	"context"

	"queueb.org/keyring-winbridge/internal/bridge"
)

type unsupportedBackend struct{}

// New returns a backend that reports Windows Credential Manager as
// unavailable on non-Windows systems.
func New() bridge.Backend {
	return &unsupportedBackend{}
}

func (*unsupportedBackend) Probe(context.Context) error {
	return bridge.ErrUnavailable
}

func (*unsupportedBackend) Get(context.Context, string, string) ([]byte, error) {
	return nil, bridge.ErrUnavailable
}

func (*unsupportedBackend) Set(context.Context, string, string, []byte) error {
	return bridge.ErrUnavailable
}

func (*unsupportedBackend) Delete(context.Context, string, string) (bool, error) {
	return false, bridge.ErrUnavailable
}
