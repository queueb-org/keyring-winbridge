package bridge

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"

	"queueb.org/keyring-winbridge/protocol/v1alpha1"
)

func TestHandlerProbe(t *testing.T) {
	handler := NewHandler(&fakeBackend{}, "0.1.0")
	request := requestFor(v1alpha1.OperationProbe)

	response := handler.Handle(t.Context(), request)
	assertValidResponse(t, request, response)
	if !response.OK || response.Result.HelperVersion != "0.1.0" {
		t.Fatalf("Handle() = %#v", response)
	}
}

func TestHandlerGet(t *testing.T) {
	handler := NewHandler(&fakeBackend{getSecret: []byte{0, 1, 2, 255}}, "0.1.0")
	request := requestFor(v1alpha1.OperationGet)

	response := handler.Handle(t.Context(), request)
	assertValidResponse(t, request, response)
	want := base64.StdEncoding.EncodeToString([]byte{0, 1, 2, 255})
	if response.Result.SecretBase64 == nil || *response.Result.SecretBase64 != want {
		t.Fatalf("secret_b64 = %#v, want %q", response.Result.SecretBase64, want)
	}
}

func TestHandlerSet(t *testing.T) {
	backend := &fakeBackend{}
	handler := NewHandler(backend, "0.1.0")
	request := requestFor(v1alpha1.OperationSet)
	secret := base64.StdEncoding.EncodeToString([]byte{0, 1, 2, 255})
	request.SecretBase64 = &secret

	response := handler.Handle(t.Context(), request)
	assertValidResponse(t, request, response)
	if !response.OK {
		t.Fatalf("Handle() = %#v", response)
	}
	if got, want := string(backend.setSecret), string([]byte{0, 1, 2, 255}); got != want {
		t.Fatalf("backend secret = %q, want %q", got, want)
	}
}

func TestHandlerDelete(t *testing.T) {
	tests := []struct {
		name    string
		backend *fakeBackend
		deleted bool
	}{
		{name: "deleted", backend: &fakeBackend{deleteResult: true}, deleted: true},
		{name: "missing", backend: &fakeBackend{deleteErr: ErrNotFound}, deleted: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(tt.backend, "0.1.0")
			request := requestFor(v1alpha1.OperationDelete)
			response := handler.Handle(t.Context(), request)
			assertValidResponse(t, request, response)
			if !response.OK || response.Result.Deleted == nil || *response.Result.Deleted != tt.deleted {
				t.Fatalf("Handle() = %#v, want deleted %t", response, tt.deleted)
			}
		})
	}
}

func TestHandlerMapsBackendErrors(t *testing.T) {
	tests := []struct {
		name       string
		backendErr error
		code       v1alpha1.ErrorCode
		retryable  bool
	}{
		{name: "not found", backendErr: ErrNotFound, code: v1alpha1.ErrNotFound},
		{name: "invalid argument", backendErr: ErrInvalidArgument, code: v1alpha1.ErrInvalidArgument},
		{name: "access denied", backendErr: ErrAccessDenied, code: v1alpha1.ErrAccessDenied},
		{name: "unavailable", backendErr: ErrUnavailable, code: v1alpha1.ErrBackendUnavailable, retryable: true},
		{name: "deadline", backendErr: context.DeadlineExceeded, code: v1alpha1.ErrTimeout, retryable: true},
		{name: "cancelled", backendErr: context.Canceled, code: v1alpha1.ErrTimeout, retryable: true},
		{name: "unclassified", backendErr: errors.New("secret backend detail"), code: v1alpha1.ErrBackendFailure},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(&fakeBackend{getErr: tt.backendErr}, "0.1.0")
			request := requestFor(v1alpha1.OperationGet)
			response := handler.Handle(t.Context(), request)
			assertValidResponse(t, request, response)
			if response.OK || response.Error.Code != tt.code || response.Error.Retryable != tt.retryable {
				t.Fatalf("Handle() = %#v", response)
			}
			if response.Error.Message == tt.backendErr.Error() {
				t.Fatal("backend error text was exposed on the wire")
			}
		})
	}
}

func TestHandlerRejectsInvalidRequestWithoutCallingBackend(t *testing.T) {
	backend := &fakeBackend{}
	handler := NewHandler(backend, "0.1.0")
	request := requestFor(v1alpha1.Operation("list"))

	response := handler.Handle(t.Context(), request)
	if response.OK || response.Error.Code != v1alpha1.ErrUnsupportedOperation {
		t.Fatalf("Handle() = %#v", response)
	}
	if backend.calls != 0 {
		t.Fatalf("backend calls = %d, want 0", backend.calls)
	}
}

func TestHandlerWithoutBackend(t *testing.T) {
	handler := NewHandler(nil, "0.1.0")
	request := requestFor(v1alpha1.OperationProbe)

	response := handler.Handle(t.Context(), request)
	assertValidResponse(t, request, response)
	if response.OK || response.Error.Code != v1alpha1.ErrInternal {
		t.Fatalf("Handle() = %#v", response)
	}
}

func requestFor(operation v1alpha1.Operation) v1alpha1.Request {
	request := v1alpha1.Request{
		Protocol:  v1alpha1.ProtocolID,
		RequestID: "request-1",
		Operation: operation,
	}
	if operation != v1alpha1.OperationProbe {
		request.Service = "example"
		request.Account = "token"
	}
	return request
}

func assertValidResponse(t *testing.T, request v1alpha1.Request, response v1alpha1.Response) {
	t.Helper()
	if err := response.ValidateFor(request); err != nil {
		t.Fatalf("ValidateFor() error = %v; response = %#v", err, response)
	}
}

type fakeBackend struct {
	probeErr     error
	getSecret    []byte
	getErr       error
	setSecret    []byte
	setErr       error
	deleteResult bool
	deleteErr    error
	calls        int
}

func (b *fakeBackend) Probe(context.Context) error {
	b.calls++
	return b.probeErr
}

func (b *fakeBackend) Get(context.Context, string, string) ([]byte, error) {
	b.calls++
	return b.getSecret, b.getErr
}

func (b *fakeBackend) Set(_ context.Context, _, _ string, secret []byte) error {
	b.calls++
	b.setSecret = append([]byte(nil), secret...)
	return b.setErr
}

func (b *fakeBackend) Delete(context.Context, string, string) (bool, error) {
	b.calls++
	return b.deleteResult, b.deleteErr
}
