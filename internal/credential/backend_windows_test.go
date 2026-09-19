//go:build windows

package credential

import (
	"errors"
	"testing"

	"github.com/danieljoos/wincred"
	"golang.org/x/sys/windows"

	"queueb.org/keyring-winbridge/internal/bridge"
)

func TestProbe(t *testing.T) {
	if err := New().Probe(t.Context()); err != nil {
		t.Fatalf("Probe() error = %v", err)
	}
}

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error
	}{
		{name: "not found", err: wincred.ErrElementNotFound, want: bridge.ErrNotFound},
		{name: "invalid parameter", err: wincred.ErrInvalidParameter, want: bridge.ErrInvalidArgument},
		{name: "bad username", err: wincred.ErrBadUsername, want: bridge.ErrInvalidArgument},
		{name: "access denied", err: windows.ERROR_ACCESS_DENIED, want: bridge.ErrAccessDenied},
		{name: "no logon session", err: windows.ERROR_NO_SUCH_LOGON_SESSION, want: bridge.ErrUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyError(tt.err); !errors.Is(got, tt.want) {
				t.Fatalf("classifyError() = %v, want wrapped %v", got, tt.want)
			}
		})
	}
}

func TestClassifyErrorPreservesUnknownError(t *testing.T) {
	want := errors.New("unknown")
	if got := classifyError(want); !errors.Is(got, want) {
		t.Fatalf("classifyError() = %v, want %v", got, want)
	}
}
