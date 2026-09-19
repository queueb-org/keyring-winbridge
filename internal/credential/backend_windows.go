//go:build windows

package credential

import (
	"context"
	"errors"
	"fmt"

	"github.com/danieljoos/wincred"
	"golang.org/x/sys/windows"

	"queueb.org/keyring-winbridge/internal/bridge"
)

var advapi32 = windows.NewLazySystemDLL("advapi32.dll")

type windowsBackend struct{}

// New returns a Windows Credential Manager backend.
func New() bridge.Backend {
	return &windowsBackend{}
}

func (*windowsBackend) Probe(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := advapi32.Load(); err != nil {
		return fmt.Errorf("%w: load advapi32.dll: %v", bridge.ErrUnavailable, err)
	}
	for _, name := range []string{"CredReadW", "CredWriteW", "CredDeleteW"} {
		if err := advapi32.NewProc(name).Find(); err != nil {
			return fmt.Errorf("%w: find %s: %v", bridge.ErrUnavailable, name, err)
		}
	}
	return nil
}

func (*windowsBackend) Get(ctx context.Context, service, account string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	credential, err := wincred.GetGenericCredential(targetName(service, account))
	if err != nil {
		return nil, classifyError(err)
	}
	return append([]byte(nil), credential.CredentialBlob...), nil
}

func (*windowsBackend) Set(
	ctx context.Context,
	service string,
	account string,
	secret []byte,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	credential := wincred.NewGenericCredential(targetName(service, account))
	credential.UserName = account
	credential.CredentialBlob = append([]byte(nil), secret...)
	if err := credential.Write(); err != nil {
		return classifyError(err)
	}
	return nil
}

func (*windowsBackend) Delete(ctx context.Context, service, account string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	credential, err := wincred.GetGenericCredential(targetName(service, account))
	if errors.Is(err, wincred.ErrElementNotFound) {
		return false, nil
	}
	if err != nil {
		return false, classifyError(err)
	}
	if err := credential.Delete(); err != nil {
		return false, classifyError(err)
	}
	return true, nil
}

func classifyError(err error) error {
	switch {
	case errors.Is(err, wincred.ErrElementNotFound):
		return fmt.Errorf("%w: %v", bridge.ErrNotFound, err)
	case errors.Is(err, wincred.ErrInvalidParameter), errors.Is(err, wincred.ErrBadUsername):
		return fmt.Errorf("%w: %v", bridge.ErrInvalidArgument, err)
	case errors.Is(err, windows.ERROR_ACCESS_DENIED):
		return fmt.Errorf("%w: %v", bridge.ErrAccessDenied, err)
	case errors.Is(err, windows.ERROR_NO_SUCH_LOGON_SESSION):
		return fmt.Errorf("%w: %v", bridge.ErrUnavailable, err)
	default:
		return err
	}
}
