package v1alpha1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"unicode/utf8"
)

// ReadRequest reads, decodes, and validates exactly one request. When
// possible, it returns envelope fields decoded before an error was found so a
// caller can include them in an error response.
func ReadRequest(reader io.Reader) (Request, error) {
	contents, err := readMessage(reader)
	if err != nil {
		return Request{}, err
	}

	var request Request
	if err := decodeMessage(contents, &request); err != nil {
		return request, err
	}
	if err := request.Validate(); err != nil {
		return request, err
	}
	return request, nil
}

// ReadResponse reads, decodes, and validates exactly one response for request.
func ReadResponse(reader io.Reader, request Request) (Response, error) {
	contents, err := readMessage(reader)
	if err != nil {
		return Response{}, err
	}

	var response Response
	if err := decodeMessage(contents, &response); err != nil {
		return response, err
	}
	if err := response.ValidateFor(request); err != nil {
		return response, err
	}
	return response, nil
}

// WriteRequest validates and writes exactly one newline-terminated request.
func WriteRequest(writer io.Writer, request Request) error {
	if err := request.Validate(); err != nil {
		return err
	}
	return writeMessage(writer, request)
}

// WriteResponse validates and writes exactly one newline-terminated response.
// Operation-specific validation remains the responsibility of the response
// producer, which has the originating request.
func WriteResponse(writer io.Writer, response Response) error {
	if err := response.Validate(); err != nil {
		return err
	}
	return writeMessage(writer, response)
}

func readMessage(reader io.Reader) ([]byte, error) {
	contents, err := io.ReadAll(io.LimitReader(reader, MaxMessageBytes+1))
	if err != nil {
		return nil, protocolError(ErrInvalidRequest, fmt.Sprintf("cannot read message: %v", err))
	}
	if len(contents) > MaxMessageBytes {
		return nil, protocolError(
			ErrInvalidRequest,
			fmt.Sprintf("message exceeds %d bytes", MaxMessageBytes),
		)
	}
	if len(bytes.TrimSpace(contents)) == 0 {
		return nil, protocolError(ErrInvalidRequest, "message is empty")
	}
	if !utf8.Valid(contents) {
		return nil, protocolError(ErrInvalidRequest, "message is not valid UTF-8")
	}
	return contents, nil
}

func decodeMessage(contents []byte, target any) error {
	if bytes.Equal(bytes.TrimSpace(contents), []byte("null")) {
		return protocolError(ErrInvalidRequest, "message must be a JSON object")
	}
	if err := json.Unmarshal(contents, target); err != nil {
		return protocolError(ErrInvalidRequest, fmt.Sprintf("invalid JSON message: %v", err))
	}
	return nil
}

func writeMessage(writer io.Writer, message any) error {
	contents, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("encode protocol message: %w", err)
	}
	contents = append(contents, '\n')
	if len(contents) > MaxMessageBytes {
		return fmt.Errorf("encoded protocol message exceeds %d bytes", MaxMessageBytes)
	}
	written, err := writer.Write(contents)
	if err != nil {
		return fmt.Errorf("write protocol message: %w", err)
	}
	if written != len(contents) {
		return fmt.Errorf("write protocol message: %w", io.ErrShortWrite)
	}
	return nil
}
