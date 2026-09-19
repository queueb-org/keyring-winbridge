package v1alpha1

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestReadRequest(t *testing.T) {
	request, err := ReadRequest(strings.NewReader(`{
  "protocol": "keyring-winbridge/v1alpha1",
  "request_id": "request-1",
  "operation": "get",
  "service": "example",
  "account": "token",
  "future_field": true
}`))
	if err != nil {
		t.Fatalf("ReadRequest() error = %v", err)
	}
	if request.Operation != OperationGet || request.Service != "example" || request.Account != "token" {
		t.Fatalf("ReadRequest() = %#v", request)
	}
}

func TestReadRequestPreservesEmptySecret(t *testing.T) {
	request, err := ReadRequest(strings.NewReader(`{
  "protocol": "keyring-winbridge/v1alpha1",
  "request_id": "request-1",
  "operation": "set",
  "service": "example",
  "account": "token",
  "secret_b64": ""
}`))
	if err != nil {
		t.Fatalf("ReadRequest() error = %v", err)
	}
	if request.SecretBase64 == nil || *request.SecretBase64 != "" {
		t.Fatalf("SecretBase64 = %#v, want pointer to empty string", request.SecretBase64)
	}
}

func TestReadRequestRejectsInvalidMessages(t *testing.T) {
	tests := []struct {
		name    string
		message []byte
	}{
		{name: "empty", message: nil},
		{name: "null", message: []byte("null")},
		{name: "invalid UTF-8", message: []byte{'{', '}', 0xff}},
		{name: "multiple objects", message: []byte("{}\n{}")},
		{name: "wrong field type", message: []byte(`{"protocol": 1}`)},
		{name: "too large", message: bytes.Repeat([]byte{' '}, MaxMessageBytes+1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ReadRequest(bytes.NewReader(tt.message))
			assertProtocolError(t, err, ErrInvalidRequest)
		})
	}
}

func TestReadRequestReturnsRecoverableEnvelope(t *testing.T) {
	request, err := ReadRequest(strings.NewReader(`{
  "protocol": "keyring-winbridge/v1alpha1",
  "request_id": "request-1",
  "operation": "list"
}`))
	assertProtocolError(t, err, ErrUnsupportedOperation)
	if request.Protocol != ProtocolID || request.RequestID != "request-1" {
		t.Fatalf("ReadRequest() = %#v, want recoverable envelope", request)
	}
}

func TestWriteRequestRoundTrip(t *testing.T) {
	secret := base64.StdEncoding.EncodeToString([]byte("secret"))
	want := Request{
		Protocol:     ProtocolID,
		RequestID:    "request-1",
		Operation:    OperationSet,
		Service:      "example",
		Account:      "token",
		SecretBase64: &secret,
	}

	var output bytes.Buffer
	if err := WriteRequest(&output, want); err != nil {
		t.Fatalf("WriteRequest() error = %v", err)
	}
	if !bytes.HasSuffix(output.Bytes(), []byte{'\n'}) {
		t.Fatalf("WriteRequest() output is not newline terminated: %q", output.Bytes())
	}

	got, err := ReadRequest(&output)
	if err != nil {
		t.Fatalf("ReadRequest() error = %v", err)
	}
	if got.SecretBase64 == nil || *got.SecretBase64 != secret {
		t.Fatalf("round trip secret = %#v, want %q", got.SecretBase64, secret)
	}
}

func TestReadResponse(t *testing.T) {
	request := Request{
		Protocol:  ProtocolID,
		RequestID: "request-1",
		Operation: OperationDelete,
		Service:   "example",
		Account:   "token",
	}
	response, err := ReadResponse(strings.NewReader(`{
  "protocol": "keyring-winbridge/v1alpha1",
  "request_id": "request-1",
  "ok": true,
  "result": {"deleted": false}
}`), request)
	if err != nil {
		t.Fatalf("ReadResponse() error = %v", err)
	}
	if response.Result == nil || response.Result.Deleted == nil || *response.Result.Deleted {
		t.Fatalf("deleted = %#v, want false", response.Result)
	}
}

func TestReadResponseRejectsMismatchedRequest(t *testing.T) {
	request := Request{Protocol: ProtocolID, RequestID: "request-1", Operation: OperationSet}
	_, err := ReadResponse(strings.NewReader(`{
  "protocol": "keyring-winbridge/v1alpha1",
  "request_id": "other",
  "ok": true,
  "result": {}
}`), request)
	assertProtocolError(t, err, ErrInvalidRequest)
}

func TestWriteResponseRejectsInvalidEnvelope(t *testing.T) {
	var output bytes.Buffer
	err := WriteResponse(&output, Response{Protocol: ProtocolID, OK: true})
	assertProtocolError(t, err, ErrInvalidRequest)
}

func TestWriteResponseRejectsOversizedOutput(t *testing.T) {
	response := Response{
		Protocol: ProtocolID,
		Error: &ProtocolError{
			Code:    ErrInternal,
			Message: strings.Repeat("x", MaxMessageBytes),
		},
	}
	var output bytes.Buffer
	if err := WriteResponse(&output, response); err == nil {
		t.Fatal("WriteResponse() error = nil, want oversized output error")
	}
}

func TestWriteResponsePropagatesWriterError(t *testing.T) {
	want := errors.New("writer failed")
	err := WriteResponse(errorWriter{err: want}, Response{
		Protocol: ProtocolID,
		Error: &ProtocolError{
			Code:    ErrInternal,
			Message: "internal failure",
		},
	})
	if !errors.Is(err, want) {
		t.Fatalf("WriteResponse() error = %v, want wrapped %v", err, want)
	}
}

func TestWriteResponseRejectsShortWrite(t *testing.T) {
	err := WriteResponse(shortWriter{}, Response{
		Protocol: ProtocolID,
		Error: &ProtocolError{
			Code:    ErrInternal,
			Message: "internal failure",
		},
	})
	if !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("WriteResponse() error = %v, want %v", err, io.ErrShortWrite)
	}
}

type errorWriter struct {
	err error
}

func (w errorWriter) Write([]byte) (int, error) {
	return 0, w.err
}

type shortWriter struct{}

func (shortWriter) Write(contents []byte) (int, error) {
	return len(contents) - 1, nil
}
