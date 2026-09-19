package bridge

import (
	"context"
	"io"

	"queueb.org/keyring-winbridge/protocol/v1alpha1"
)

const (
	// ExitSuccess indicates that an operation completed successfully.
	ExitSuccess = 0
	// ExitOperationError indicates that a valid protocol error was returned.
	ExitOperationError = 1
	// ExitTransportError indicates that no valid response could be written.
	ExitTransportError = 2
)

// Run reads, dispatches, and writes exactly one v1alpha1 exchange.
func Run(
	ctx context.Context,
	input io.Reader,
	output io.Writer,
	handler *Handler,
) int {
	request, err := v1alpha1.ReadRequest(input)
	if err != nil {
		response := failure(request, asProtocolError(err))
		if err := v1alpha1.WriteResponse(output, response); err != nil {
			return ExitTransportError
		}
		return ExitOperationError
	}

	response := handler.Handle(ctx, request)
	if err := response.ValidateFor(request); err != nil {
		return ExitTransportError
	}
	if err := v1alpha1.WriteResponse(output, response); err != nil {
		return ExitTransportError
	}
	if response.OK {
		return ExitSuccess
	}
	return ExitOperationError
}
