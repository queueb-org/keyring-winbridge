package bridge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"queueb.org/keyring-winbridge/protocol/v1alpha1"
)

func TestRunSuccess(t *testing.T) {
	request := requestFor(v1alpha1.OperationProbe)
	var input bytes.Buffer
	if err := v1alpha1.WriteRequest(&input, request); err != nil {
		t.Fatalf("WriteRequest() error = %v", err)
	}

	var output bytes.Buffer
	exitCode := Run(
		t.Context(),
		&input,
		&output,
		NewHandler(&fakeBackend{}, "0.1.0"),
	)
	if exitCode != ExitSuccess {
		t.Fatalf("Run() = %d, want %d", exitCode, ExitSuccess)
	}

	response, err := v1alpha1.ReadResponse(&output, request)
	if err != nil {
		t.Fatalf("ReadResponse() error = %v", err)
	}
	if !response.OK {
		t.Fatalf("response = %#v", response)
	}
}

func TestRunOperationError(t *testing.T) {
	request := requestFor(v1alpha1.OperationGet)
	var input bytes.Buffer
	if err := v1alpha1.WriteRequest(&input, request); err != nil {
		t.Fatalf("WriteRequest() error = %v", err)
	}

	var output bytes.Buffer
	exitCode := Run(
		t.Context(),
		&input,
		&output,
		NewHandler(&fakeBackend{getErr: ErrNotFound}, "0.1.0"),
	)
	if exitCode != ExitOperationError {
		t.Fatalf("Run() = %d, want %d", exitCode, ExitOperationError)
	}

	response, err := v1alpha1.ReadResponse(&output, request)
	if err != nil {
		t.Fatalf("ReadResponse() error = %v", err)
	}
	if response.OK || response.Error.Code != v1alpha1.ErrNotFound {
		t.Fatalf("response = %#v", response)
	}
}

func TestRunInvalidJSON(t *testing.T) {
	var output bytes.Buffer
	exitCode := Run(
		t.Context(),
		bytes.NewBufferString(`{"protocol":`),
		&output,
		NewHandler(&fakeBackend{}, "0.1.0"),
	)
	if exitCode != ExitOperationError {
		t.Fatalf("Run() = %d, want %d", exitCode, ExitOperationError)
	}

	var response v1alpha1.Response
	if err := json.Unmarshal(output.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if response.OK || response.Protocol != v1alpha1.ProtocolID || response.Error.Code != v1alpha1.ErrInvalidRequest {
		t.Fatalf("response = %#v", response)
	}
}

func TestRunUnsupportedProtocol(t *testing.T) {
	input := bytes.NewBufferString(`{
  "protocol": "keyring-winbridge/v2",
  "request_id": "request-1",
  "operation": "probe"
}`)
	var output bytes.Buffer
	exitCode := Run(t.Context(), input, &output, NewHandler(&fakeBackend{}, "0.1.0"))
	if exitCode != ExitOperationError {
		t.Fatalf("Run() = %d, want %d", exitCode, ExitOperationError)
	}

	var response v1alpha1.Response
	if err := json.Unmarshal(output.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if response.Protocol != "keyring-winbridge/v2" || response.Error.Code != v1alpha1.ErrUnsupportedProtocol {
		t.Fatalf("response = %#v", response)
	}
	if len(response.Error.SupportedProtocols) != 1 || response.Error.SupportedProtocols[0] != v1alpha1.ProtocolID {
		t.Fatalf("supported protocols = %v", response.Error.SupportedProtocols)
	}
}

func TestRunWriterFailure(t *testing.T) {
	request := requestFor(v1alpha1.OperationProbe)
	var input bytes.Buffer
	if err := v1alpha1.WriteRequest(&input, request); err != nil {
		t.Fatalf("WriteRequest() error = %v", err)
	}

	exitCode := Run(
		t.Context(),
		&input,
		errorWriter{err: errors.New("writer failed")},
		NewHandler(&fakeBackend{}, "0.1.0"),
	)
	if exitCode != ExitTransportError {
		t.Fatalf("Run() = %d, want %d", exitCode, ExitTransportError)
	}
}

func TestRunCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	request := requestFor(v1alpha1.OperationGet)
	var input bytes.Buffer
	if err := v1alpha1.WriteRequest(&input, request); err != nil {
		t.Fatalf("WriteRequest() error = %v", err)
	}

	var output bytes.Buffer
	exitCode := Run(
		ctx,
		&input,
		&output,
		NewHandler(&fakeBackend{getErr: context.Canceled}, "0.1.0"),
	)
	if exitCode != ExitOperationError {
		t.Fatalf("Run() = %d, want %d", exitCode, ExitOperationError)
	}
}

type errorWriter struct {
	err error
}

func (w errorWriter) Write([]byte) (int, error) {
	return 0, w.err
}
