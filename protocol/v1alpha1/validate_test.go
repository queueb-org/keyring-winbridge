package v1alpha1

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRequestFixtures(t *testing.T) {
	tests := []struct {
		name string
		file string
	}{
		{name: "probe", file: "request-probe.json"},
		{name: "get", file: "request-get.json"},
		{name: "set", file: "request-set.json"},
		{name: "set empty secret", file: "request-set-empty.json"},
		{name: "delete", file: "request-delete.json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var request Request
			readFixture(t, tt.file, &request)
			if err := request.Validate(); err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}

func TestResponseFixtures(t *testing.T) {
	tests := []struct {
		name         string
		requestFile  string
		responseFile string
	}{
		{name: "probe", requestFile: "request-probe.json", responseFile: "response-probe.json"},
		{name: "get", requestFile: "request-get.json", responseFile: "response-get.json"},
		{name: "get empty secret", requestFile: "request-get.json", responseFile: "response-get-empty.json"},
		{name: "set", requestFile: "request-set.json", responseFile: "response-set.json"},
		{name: "set empty secret", requestFile: "request-set-empty.json", responseFile: "response-set.json"},
		{name: "delete", requestFile: "request-delete.json", responseFile: "response-delete.json"},
		{name: "delete missing", requestFile: "request-delete.json", responseFile: "response-delete-missing.json"},
		{name: "not found", requestFile: "request-get.json", responseFile: "response-not-found.json"},
		{name: "unsupported protocol", requestFile: "request-probe-unsupported.json", responseFile: "response-unsupported-protocol.json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var request Request
			readFixture(t, tt.requestFile, &request)

			var response Response
			readFixture(t, tt.responseFile, &response)
			if err := response.ValidateFor(request); err != nil {
				t.Fatalf("ValidateFor() error = %v", err)
			}
		})
	}
}

func TestRequestValidateRejectsInvalidRequests(t *testing.T) {
	secret := base64.StdEncoding.EncodeToString([]byte("secret"))
	oversized := base64.StdEncoding.EncodeToString(make([]byte, MaxSecretBytes+1))

	tests := []struct {
		name    string
		request Request
		code    ErrorCode
	}{
		{
			name:    "unsupported protocol",
			request: Request{Protocol: "keyring-winbridge/v2", RequestID: "1", Operation: OperationProbe},
			code:    ErrUnsupportedProtocol,
		},
		{
			name:    "missing request id",
			request: Request{Protocol: ProtocolID, Operation: OperationProbe},
			code:    ErrInvalidRequest,
		},
		{
			name:    "unsupported operation",
			request: Request{Protocol: ProtocolID, RequestID: "1", Operation: "list"},
			code:    ErrUnsupportedOperation,
		},
		{
			name:    "probe fields",
			request: Request{Protocol: ProtocolID, RequestID: "1", Operation: OperationProbe, Service: "service"},
			code:    ErrInvalidRequest,
		},
		{
			name:    "missing account",
			request: Request{Protocol: ProtocolID, RequestID: "1", Operation: OperationGet, Service: "service"},
			code:    ErrInvalidArgument,
		},
		{
			name:    "unexpected secret",
			request: Request{Protocol: ProtocolID, RequestID: "1", Operation: OperationGet, Service: "service", Account: "account", SecretBase64: &secret},
			code:    ErrInvalidRequest,
		},
		{
			name:    "missing set secret",
			request: Request{Protocol: ProtocolID, RequestID: "1", Operation: OperationSet, Service: "service", Account: "account"},
			code:    ErrInvalidRequest,
		},
		{
			name:    "invalid base64",
			request: Request{Protocol: ProtocolID, RequestID: "1", Operation: OperationSet, Service: "service", Account: "account", SecretBase64: stringPointer("not base64")},
			code:    ErrInvalidRequest,
		},
		{
			name:    "oversized secret",
			request: Request{Protocol: ProtocolID, RequestID: "1", Operation: OperationSet, Service: "service", Account: "account", SecretBase64: &oversized},
			code:    ErrInvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			assertProtocolError(t, err, tt.code)
		})
	}
}

func TestUnsupportedProtocolAdvertisesSupportedProtocols(t *testing.T) {
	request := Request{Protocol: "keyring-winbridge/v2", RequestID: "1", Operation: OperationProbe}

	err := request.Validate()
	var protocolErr *ProtocolError
	if !errors.As(err, &protocolErr) {
		t.Fatalf("error type = %T, want *ProtocolError", err)
	}
	if len(protocolErr.SupportedProtocols) != 1 || protocolErr.SupportedProtocols[0] != ProtocolID {
		t.Fatalf("supported protocols = %v, want [%q]", protocolErr.SupportedProtocols, ProtocolID)
	}
}

func TestResponseValidateRejectsInvalidResponses(t *testing.T) {
	request := Request{
		Protocol:  ProtocolID,
		RequestID: "request-1",
		Operation: OperationDelete,
		Service:   "service",
		Account:   "account",
	}
	deleted := true

	tests := []struct {
		name     string
		response Response
	}{
		{
			name:     "mismatched request id",
			response: Response{Protocol: ProtocolID, RequestID: "other", OK: true, Result: &Result{Deleted: &deleted}},
		},
		{
			name:     "missing result",
			response: Response{Protocol: ProtocolID, RequestID: request.RequestID, OK: true},
		},
		{
			name: "success with error",
			response: Response{
				Protocol:  ProtocolID,
				RequestID: request.RequestID,
				OK:        true,
				Result:    &Result{Deleted: &deleted},
				Error:     &ProtocolError{Code: ErrInternal, Message: "failure"},
			},
		},
		{
			name:     "missing deleted",
			response: Response{Protocol: ProtocolID, RequestID: request.RequestID, OK: true, Result: &Result{}},
		},
		{
			name:     "unknown error code",
			response: Response{Protocol: ProtocolID, RequestID: request.RequestID, Error: &ProtocolError{Code: "unknown", Message: "failure"}},
		},
		{
			name: "protocol list on unrelated error",
			response: Response{
				Protocol:  ProtocolID,
				RequestID: request.RequestID,
				Error: &ProtocolError{
					Code:               ErrInternal,
					Message:            "failure",
					SupportedProtocols: []string{ProtocolID},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.response.ValidateFor(request)
			assertProtocolError(t, err, ErrInvalidRequest)
		})
	}
}

func TestProtocolErrorImplementsError(t *testing.T) {
	err := error(&ProtocolError{Code: ErrNotFound, Message: "credential was not found"})
	if got, want := err.Error(), "not_found: credential was not found"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

func TestStrictBase64RejectsNonZeroTrailingBits(t *testing.T) {
	request := Request{
		Protocol:     ProtocolID,
		RequestID:    "request-1",
		Operation:    OperationSet,
		Service:      "service",
		Account:      "account",
		SecretBase64: stringPointer("Zh=="),
	}

	err := request.Validate()
	assertProtocolError(t, err, ErrInvalidRequest)
}

func TestCredentialIDRejectsNUL(t *testing.T) {
	request := Request{
		Protocol:  ProtocolID,
		RequestID: "request-1",
		Operation: OperationGet,
		Service:   "service",
		Account:   strings.Join([]string{"api", "token"}, "\x00"),
	}

	err := request.Validate()
	assertProtocolError(t, err, ErrInvalidArgument)
}

func readFixture(t *testing.T, name string, target any) {
	t.Helper()

	contents, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if len(contents) > MaxMessageBytes {
		t.Fatalf("fixture exceeds MaxMessageBytes: %d", len(contents))
	}
	if err := json.Unmarshal(contents, target); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
}

func assertProtocolError(t *testing.T, err error, code ErrorCode) {
	t.Helper()

	if err == nil {
		t.Fatal("expected an error")
	}
	var protocolErr *ProtocolError
	if !errors.As(err, &protocolErr) {
		t.Fatalf("error type = %T, want *ProtocolError", err)
	}
	if protocolErr.Code != code {
		t.Fatalf("error code = %q, want %q", protocolErr.Code, code)
	}
}

func stringPointer(value string) *string {
	return &value
}
