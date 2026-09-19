// Package bridge dispatches validated protocol requests to a credential
// backend without exposing backend-specific errors on the wire.
package bridge

import (
	"context"
	"encoding/base64"
	"errors"

	"queueb.org/keyring-winbridge/protocol/v1alpha1"
)

var (
	// ErrNotFound reports that a requested credential does not exist.
	ErrNotFound = errors.New("credential not found")
	// ErrInvalidArgument reports that the backend rejected an identifier or
	// secret accepted by the protocol layer.
	ErrInvalidArgument = errors.New("invalid backend argument")
	// ErrAccessDenied reports that Windows denied access to a credential.
	ErrAccessDenied = errors.New("credential access denied")
	// ErrUnavailable reports that the credential backend cannot be reached.
	ErrUnavailable = errors.New("credential backend unavailable")
)

// Backend is the storage contract required by the protocol dispatcher.
type Backend interface {
	Probe(context.Context) error
	Get(context.Context, string, string) ([]byte, error)
	Set(context.Context, string, string, []byte) error
	Delete(context.Context, string, string) (bool, error)
}

// Handler dispatches v1alpha1 requests to a Backend.
type Handler struct {
	backend       Backend
	helperVersion string
}

// NewHandler creates a protocol handler. helperVersion is reported by probe
// and should be the executable release version.
func NewHandler(backend Backend, helperVersion string) *Handler {
	if helperVersion == "" {
		helperVersion = "dev"
	}
	return &Handler{backend: backend, helperVersion: helperVersion}
}

// Handle validates and executes one request.
func (h *Handler) Handle(ctx context.Context, request v1alpha1.Request) v1alpha1.Response {
	if err := request.Validate(); err != nil {
		return failure(request, asProtocolError(err))
	}
	if h == nil || h.backend == nil {
		return failure(request, &v1alpha1.ProtocolError{
			Code:    v1alpha1.ErrInternal,
			Message: "credential backend is not configured",
		})
	}

	switch request.Operation {
	case v1alpha1.OperationProbe:
		if err := h.backend.Probe(ctx); err != nil {
			return failure(request, backendError(err))
		}
		return success(request, &v1alpha1.Result{
			HelperVersion:      h.helperVersion,
			SupportedProtocols: []string{v1alpha1.ProtocolID},
			Backend:            v1alpha1.BackendWindowsCredentialManager,
		})
	case v1alpha1.OperationGet:
		secret, err := h.backend.Get(ctx, request.Service, request.Account)
		if err != nil {
			return failure(request, backendError(err))
		}
		encoded := base64.StdEncoding.EncodeToString(secret)
		return success(request, &v1alpha1.Result{SecretBase64: &encoded})
	case v1alpha1.OperationSet:
		secret, err := base64.StdEncoding.Strict().DecodeString(*request.SecretBase64)
		if err != nil {
			return failure(request, &v1alpha1.ProtocolError{
				Code:    v1alpha1.ErrInternal,
				Message: "validated secret could not be decoded",
			})
		}
		if err := h.backend.Set(ctx, request.Service, request.Account, secret); err != nil {
			return failure(request, backendError(err))
		}
		return success(request, &v1alpha1.Result{})
	case v1alpha1.OperationDelete:
		deleted, err := h.backend.Delete(ctx, request.Service, request.Account)
		if errors.Is(err, ErrNotFound) {
			deleted, err = false, nil
		}
		if err != nil {
			return failure(request, backendError(err))
		}
		return success(request, &v1alpha1.Result{Deleted: &deleted})
	default:
		return failure(request, &v1alpha1.ProtocolError{
			Code:    v1alpha1.ErrUnsupportedOperation,
			Message: "unsupported operation",
		})
	}
}

func success(request v1alpha1.Request, result *v1alpha1.Result) v1alpha1.Response {
	return v1alpha1.Response{
		Protocol:  responseProtocol(request),
		RequestID: request.RequestID,
		OK:        true,
		Result:    result,
	}
}

func failure(request v1alpha1.Request, protocolErr *v1alpha1.ProtocolError) v1alpha1.Response {
	return v1alpha1.Response{
		Protocol:  responseProtocol(request),
		RequestID: request.RequestID,
		Error:     protocolErr,
	}
}

func responseProtocol(request v1alpha1.Request) string {
	if request.Protocol != "" {
		return request.Protocol
	}
	return v1alpha1.ProtocolID
}

func asProtocolError(err error) *v1alpha1.ProtocolError {
	var protocolErr *v1alpha1.ProtocolError
	if errors.As(err, &protocolErr) {
		return protocolErr
	}
	return &v1alpha1.ProtocolError{
		Code:    v1alpha1.ErrInternal,
		Message: "unexpected protocol failure",
	}
}

func backendError(err error) *v1alpha1.ProtocolError {
	switch {
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		return &v1alpha1.ProtocolError{
			Code:      v1alpha1.ErrTimeout,
			Message:   "credential operation timed out or was cancelled",
			Retryable: true,
		}
	case errors.Is(err, ErrNotFound):
		return &v1alpha1.ProtocolError{
			Code:    v1alpha1.ErrNotFound,
			Message: "credential was not found",
		}
	case errors.Is(err, ErrInvalidArgument):
		return &v1alpha1.ProtocolError{
			Code:    v1alpha1.ErrInvalidArgument,
			Message: "credential backend rejected an argument",
		}
	case errors.Is(err, ErrAccessDenied):
		return &v1alpha1.ProtocolError{
			Code:    v1alpha1.ErrAccessDenied,
			Message: "credential access was denied",
		}
	case errors.Is(err, ErrUnavailable):
		return &v1alpha1.ProtocolError{
			Code:      v1alpha1.ErrBackendUnavailable,
			Message:   "credential backend is unavailable",
			Retryable: true,
		}
	default:
		return &v1alpha1.ProtocolError{
			Code:    v1alpha1.ErrBackendFailure,
			Message: "credential backend operation failed",
		}
	}
}
