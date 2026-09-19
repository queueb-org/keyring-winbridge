package v1alpha1

import (
	"encoding/base64"
	"fmt"
	"slices"
	"strings"
)

// Validate validates a request against the v1alpha1 contract.
func (r Request) Validate() error {
	if r.Protocol != ProtocolID {
		err := protocolError(
			ErrUnsupportedProtocol,
			fmt.Sprintf("unsupported protocol %q", r.Protocol),
		)
		err.SupportedProtocols = []string{ProtocolID}
		return err
	}
	if r.RequestID == "" {
		return protocolError(ErrInvalidRequest, "request_id is required")
	}

	switch r.Operation {
	case OperationProbe:
		if r.Service != "" || r.Account != "" || r.SecretBase64 != nil {
			return protocolError(
				ErrInvalidRequest,
				"probe does not accept credential fields",
			)
		}
	case OperationGet, OperationDelete:
		if err := validateCredentialID(r.Service, r.Account); err != nil {
			return err
		}
		if r.SecretBase64 != nil {
			return protocolError(
				ErrInvalidRequest,
				fmt.Sprintf("%s does not accept secret_b64", r.Operation),
			)
		}
	case OperationSet:
		if err := validateCredentialID(r.Service, r.Account); err != nil {
			return err
		}
		if r.SecretBase64 == nil {
			return protocolError(ErrInvalidRequest, "set requires secret_b64")
		}
		if err := validateSecret(*r.SecretBase64); err != nil {
			return err
		}
	default:
		return protocolError(
			ErrUnsupportedOperation,
			fmt.Sprintf("unsupported operation %q", r.Operation),
		)
	}

	return nil
}

// Validate validates the response envelope without operation-specific context.
func (r Response) Validate() error {
	if r.Protocol == "" {
		return protocolError(ErrInvalidRequest, "response protocol is required")
	}
	if r.OK {
		if r.Result == nil || r.Error != nil {
			return protocolError(
				ErrInvalidRequest,
				"successful response must contain only result",
			)
		}
		return nil
	}

	if r.Result != nil || r.Error == nil {
		return protocolError(
			ErrInvalidRequest,
			"failed response must contain only error",
		)
	}
	return r.Error.validate()
}

// ValidateFor validates a response against the request that produced it.
func (r Response) ValidateFor(request Request) error {
	if err := r.Validate(); err != nil {
		return err
	}
	if r.Protocol != request.Protocol {
		return protocolError(ErrInvalidRequest, "response protocol does not match request")
	}
	if r.RequestID != request.RequestID {
		return protocolError(ErrInvalidRequest, "response request_id does not match request")
	}
	if r.OK {
		return r.Result.validateFor(request.Operation)
	}
	return nil
}

func (r Result) validateFor(operation Operation) error {
	switch operation {
	case OperationProbe:
		if r.HelperVersion == "" {
			return protocolError(ErrInvalidRequest, "probe result requires helper_version")
		}
		if r.Backend != BackendWindowsCredentialManager {
			return protocolError(ErrInvalidRequest, "probe result has an invalid backend")
		}
		if !slices.Contains(r.SupportedProtocols, ProtocolID) {
			return protocolError(
				ErrInvalidRequest,
				"probe result does not include v1alpha1 support",
			)
		}
		if r.SecretBase64 != nil || r.Deleted != nil {
			return protocolError(ErrInvalidRequest, "probe result contains credential fields")
		}
	case OperationGet:
		if r.SecretBase64 == nil {
			return protocolError(ErrInvalidRequest, "get result requires secret_b64")
		}
		if err := validateSecret(*r.SecretBase64); err != nil {
			return err
		}
		if r.hasProbeFields() || r.Deleted != nil {
			return protocolError(ErrInvalidRequest, "get result contains unrelated fields")
		}
	case OperationSet:
		if r.hasProbeFields() || r.SecretBase64 != nil || r.Deleted != nil {
			return protocolError(ErrInvalidRequest, "set result must be empty")
		}
	case OperationDelete:
		if r.Deleted == nil {
			return protocolError(ErrInvalidRequest, "delete result requires deleted")
		}
		if r.hasProbeFields() || r.SecretBase64 != nil {
			return protocolError(ErrInvalidRequest, "delete result contains unrelated fields")
		}
	default:
		return protocolError(
			ErrUnsupportedOperation,
			fmt.Sprintf("unsupported operation %q", operation),
		)
	}

	return nil
}

func (r Result) hasProbeFields() bool {
	return r.HelperVersion != "" || r.SupportedProtocols != nil || r.Backend != ""
}

func (e *ProtocolError) validate() error {
	if e.Code == "" {
		return protocolError(ErrInvalidRequest, "error code is required")
	}
	if !validErrorCode(e.Code) {
		return protocolError(
			ErrInvalidRequest,
			fmt.Sprintf("unknown error code %q", e.Code),
		)
	}
	if e.Message == "" {
		return protocolError(ErrInvalidRequest, "error message is required")
	}
	if e.Code == ErrUnsupportedProtocol && len(e.SupportedProtocols) == 0 {
		return protocolError(
			ErrInvalidRequest,
			"unsupported_protocol requires supported_protocols",
		)
	}
	if e.Code != ErrUnsupportedProtocol && len(e.SupportedProtocols) != 0 {
		return protocolError(
			ErrInvalidRequest,
			"supported_protocols is only valid for unsupported_protocol",
		)
	}
	return nil
}

func validateCredentialID(service, account string) error {
	if service == "" {
		return protocolError(ErrInvalidArgument, "service is required")
	}
	if account == "" {
		return protocolError(ErrInvalidArgument, "account is required")
	}
	if strings.IndexByte(service, 0) >= 0 {
		return protocolError(ErrInvalidArgument, "service contains a NUL character")
	}
	if strings.IndexByte(account, 0) >= 0 {
		return protocolError(ErrInvalidArgument, "account contains a NUL character")
	}
	return nil
}

func validateSecret(encoded string) error {
	secret, err := base64.StdEncoding.Strict().DecodeString(encoded)
	if err != nil {
		return protocolError(ErrInvalidRequest, "secret_b64 is not valid padded Base64")
	}
	if len(secret) > MaxSecretBytes {
		return protocolError(
			ErrInvalidArgument,
			fmt.Sprintf("decoded secret exceeds %d bytes", MaxSecretBytes),
		)
	}
	return nil
}

func validErrorCode(code ErrorCode) bool {
	switch code {
	case ErrInvalidRequest,
		ErrUnsupportedProtocol,
		ErrUnsupportedOperation,
		ErrInvalidArgument,
		ErrNotFound,
		ErrAccessDenied,
		ErrBackendUnavailable,
		ErrBackendFailure,
		ErrTimeout,
		ErrInternal:
		return true
	default:
		return false
	}
}

func protocolError(code ErrorCode, message string) *ProtocolError {
	return &ProtocolError{Code: code, Message: message}
}
