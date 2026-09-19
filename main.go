package main

import (
	"context"
	"io"
	"os"
	"time"

	"queueb.org/keyring-winbridge/internal/bridge"
	"queueb.org/keyring-winbridge/internal/credential"
	v "queueb.org/keyring-winbridge/internal/version"
)

const operationTimeout = 15 * time.Second

var version = v.Version()

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
	defer cancel()

	handler := bridge.NewHandler(credential.New(), version)
	os.Exit(execute(ctx, os.Stdin, os.Stdout, handler))
}

func execute(
	ctx context.Context,
	input io.Reader,
	output io.Writer,
	handler *bridge.Handler,
) int {
	result := make(chan int, 1)
	go func() {
		result <- bridge.Run(ctx, input, output, handler)
	}()

	select {
	case exitCode := <-result:
		return exitCode
	case <-ctx.Done():
		return bridge.ExitTransportError
	}
}
