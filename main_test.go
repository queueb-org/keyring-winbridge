package main

import (
	"bytes"
	"context"
	"io"
	"testing"

	"queueb.org/keyring-winbridge/internal/bridge"
	"queueb.org/keyring-winbridge/protocol/v1alpha1"
)

func TestExecuteSuccess(t *testing.T) {
	request := v1alpha1.Request{
		Protocol:  v1alpha1.ProtocolID,
		RequestID: "request-1",
		Operation: v1alpha1.OperationProbe,
	}
	var input bytes.Buffer
	if err := v1alpha1.WriteRequest(&input, request); err != nil {
		t.Fatalf("WriteRequest() error = %v", err)
	}

	var output bytes.Buffer
	exitCode := execute(
		t.Context(),
		&input,
		&output,
		bridge.NewHandler(testBackend{}, "test"),
	)
	if exitCode != bridge.ExitSuccess {
		t.Fatalf("execute() = %d, want %d", exitCode, bridge.ExitSuccess)
	}
	if _, err := v1alpha1.ReadResponse(&output, request); err != nil {
		t.Fatalf("ReadResponse() error = %v", err)
	}
}

func TestExecuteStopsOnContextCancellation(t *testing.T) {
	release := make(chan struct{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	exitCode := execute(
		ctx,
		blockingReader{release: release},
		io.Discard,
		bridge.NewHandler(testBackend{}, "test"),
	)
	close(release)
	if exitCode != bridge.ExitTransportError {
		t.Fatalf("execute() = %d, want %d", exitCode, bridge.ExitTransportError)
	}
}

type blockingReader struct {
	release <-chan struct{}
}

func (r blockingReader) Read([]byte) (int, error) {
	<-r.release
	return 0, io.EOF
}

type testBackend struct{}

func (testBackend) Probe(context.Context) error {
	return nil
}

func (testBackend) Get(context.Context, string, string) ([]byte, error) {
	return nil, bridge.ErrNotFound
}

func (testBackend) Set(context.Context, string, string, []byte) error {
	return nil
}

func (testBackend) Delete(context.Context, string, string) (bool, error) {
	return false, nil
}
