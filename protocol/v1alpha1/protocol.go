// Package v1alpha1 defines the keyring-winbridge v1alpha1 wire protocol.
package v1alpha1

import "fmt"

const (
	// ProtocolID identifies this version of the wire protocol.
	ProtocolID = "keyring-winbridge/v1alpha1"

	// BackendWindowsCredentialManager identifies the backend exposed by the
	// bridge.
	BackendWindowsCredentialManager = "windows-credential-manager"

	// MaxMessageBytes is the largest encoded request or response accepted by
	// the protocol.
	MaxMessageBytes = 1 << 20

	// MaxSecretBytes is the Windows generic credential blob limit.
	MaxSecretBytes = 5 * 512
)

// Operation identifies an operation requested from the bridge.
type Operation string

const (
	OperationProbe  Operation = "probe"
	OperationGet    Operation = "get"
	OperationSet    Operation = "set"
	OperationDelete Operation = "delete"
)

// ErrorCode is a stable, machine-readable protocol error code.
type ErrorCode string

const (
	ErrInvalidRequest       ErrorCode = "invalid_request"
	ErrUnsupportedProtocol  ErrorCode = "unsupported_protocol"
	ErrUnsupportedOperation ErrorCode = "unsupported_operation"
	ErrInvalidArgument      ErrorCode = "invalid_argument"
	ErrNotFound             ErrorCode = "not_found"
	ErrAccessDenied         ErrorCode = "access_denied"
	ErrBackendUnavailable   ErrorCode = "backend_unavailable"
	ErrBackendFailure       ErrorCode = "backend_failure"
	ErrTimeout              ErrorCode = "timeout"
	ErrInternal             ErrorCode = "internal"
)

// Request is the request envelope for a single bridge invocation.
type Request struct {
	Protocol  string    `json:"protocol"`
	RequestID string    `json:"request_id"`
	Operation Operation `json:"operation"`
	Service   string    `json:"service,omitempty"`
	Account   string    `json:"account,omitempty"`

	// SecretBase64 is a pointer so an omitted secret can be distinguished
	// from a present, empty secret.
	SecretBase64 *string `json:"secret_b64,omitempty"`
}

// Response is the response envelope for a single bridge invocation.
type Response struct {
	Protocol  string         `json:"protocol"`
	RequestID string         `json:"request_id,omitempty"`
	OK        bool           `json:"ok"`
	Result    *Result        `json:"result,omitempty"`
	Error     *ProtocolError `json:"error,omitempty"`
}

// Result contains the operation-specific successful response fields.
type Result struct {
	HelperVersion      string   `json:"helper_version,omitempty"`
	SupportedProtocols []string `json:"supported_protocols,omitempty"`
	Backend            string   `json:"backend,omitempty"`
	SecretBase64       *string  `json:"secret_b64,omitempty"`
	Deleted            *bool    `json:"deleted,omitempty"`
}

// ProtocolError is the error object returned by the bridge. Code is stable;
// Message is diagnostic text and must not be used for program logic.
type ProtocolError struct {
	Code               ErrorCode `json:"code"`
	Message            string    `json:"message"`
	Retryable          bool      `json:"retryable"`
	SupportedProtocols []string  `json:"supported_protocols,omitempty"`
}

// Error implements error.
func (e *ProtocolError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message == "" {
		return string(e.Code)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}
