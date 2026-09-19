# keyring-winbridge examples

These examples show the `v1alpha1` protocol directly. They are useful for
installation checks, troubleshooting, and development. Once the bridge backend
is available there, normal applications should use `common.queueb.org/keyring`
instead of constructing protocol JSON themselves.

The helper reads exactly one JSON request from standard input, writes exactly
one JSON response to standard output, and exits. Secrets must not be passed in
command-line arguments or environment variables.

## Locate the helper from WSL

Build the executable from this repository:

```bash
./hack/build.sh
```

The resulting `bin/keyring-winbridge.exe` can be executed directly from WSL
when Windows executable interop is enabled. For an installed helper, keep only
its path in an environment variable:

```bash
export KEYRING_WINBRIDGE=/mnt/c/Tools/keyring-winbridge.exe
```

The environment variable contains a path, never a secret.

## Probe the helper

`probe` checks the protocol and Windows Credential Manager availability
without reading or modifying a credential:

```bash
printf '%s\n' \
  '{"protocol":"keyring-winbridge/v1alpha1","request_id":"example-probe","operation":"probe"}' \
  | "$KEYRING_WINBRIDGE"
```

Example response:

```json
{
  "protocol": "keyring-winbridge/v1alpha1",
  "request_id": "example-probe",
  "ok": true,
  "result": {
    "helper_version": "0.1.0",
    "supported_protocols": ["keyring-winbridge/v1alpha1"],
    "backend": "windows-credential-manager"
  }
}
```

## Store a demonstration secret

The Base64 value below represents the non-sensitive text `example-secret`.
Use it only for testing:

```bash
printf '%s\n' \
  '{"protocol":"keyring-winbridge/v1alpha1","request_id":"example-set","operation":"set","service":"queueb.org/keyring-winbridge/example","account":"demo-secret","secret_b64":"ZXhhbXBsZS1zZWNyZXQ="}' \
  | "$KEYRING_WINBRIDGE"
```

A successful `set` returns:

```json
{
  "protocol": "keyring-winbridge/v1alpha1",
  "request_id": "example-set",
  "ok": true,
  "result": {}
}
```

Do not substitute a real secret into this shell command. A real application
should construct the request in memory and send it through the helper's stdin.

## Read the demonstration secret

The raw response keeps the secret Base64-encoded:

```bash
printf '%s\n' \
  '{"protocol":"keyring-winbridge/v1alpha1","request_id":"example-get","operation":"get","service":"queueb.org/keyring-winbridge/example","account":"demo-secret"}' \
  | "$KEYRING_WINBRIDGE"
```

For this demonstration value only, the response can be decoded with `jq` and
GNU `base64`:

```bash
printf '%s\n' \
  '{"protocol":"keyring-winbridge/v1alpha1","request_id":"example-get","operation":"get","service":"queueb.org/keyring-winbridge/example","account":"demo-secret"}' \
  | "$KEYRING_WINBRIDGE" \
  | jq -r '.result.secret_b64' \
  | base64 --decode
```

This prints the secret to the terminal and is therefore unsuitable for real
credentials.

## Delete the demonstration credential

```bash
printf '%s\n' \
  '{"protocol":"keyring-winbridge/v1alpha1","request_id":"example-delete","operation":"delete","service":"queueb.org/keyring-winbridge/example","account":"demo-secret"}' \
  | "$KEYRING_WINBRIDGE"
```

The result contains `"deleted": true`. Repeating the request is safe and
returns `"deleted": false`.

## Handle errors and exit codes

The helper uses these process exit codes:

- `0`: the response contains `"ok": true`;
- `1`: the response contains a valid protocol error;
- `2`: no valid response could be produced.

Always parse a present response before interpreting the exit code. For
example, a missing credential returns structured JSON and exit code `1`:

```bash
set +e
response="$(printf '%s\n' \
  '{"protocol":"keyring-winbridge/v1alpha1","request_id":"example-missing","operation":"get","service":"queueb.org/keyring-winbridge/example","account":"missing"}' \
  | "$KEYRING_WINBRIDGE")"
status=$?
set -e

printf 'exit=%d response=%s\n' "$status" "$response"
```

Application code should branch on `error.code`, not on `error.message`.

## Call the helper from Go

The protocol package can encode requests and validate responses. Start the
executable directly without a shell:

```go
package example

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"

	"queueb.org/keyring-winbridge/protocol/v1alpha1"
)

func probe(ctx context.Context, helperPath string) error {
	request := v1alpha1.Request{
		Protocol:  v1alpha1.ProtocolID,
		RequestID: "example-probe",
		Operation: v1alpha1.OperationProbe,
	}

	var input bytes.Buffer
	if err := v1alpha1.WriteRequest(&input, request); err != nil {
		return err
	}

	command := exec.CommandContext(ctx, helperPath)
	command.Stdin = &input
	output, runErr := command.Output()
	if runErr != nil && len(output) == 0 {
		return fmt.Errorf("run helper: %w", runErr)
	}

	response, decodeErr := v1alpha1.ReadResponse(bytes.NewReader(output), request)
	if decodeErr != nil {
		return fmt.Errorf("invalid helper response: %w", decodeErr)
	}
	if response.OK {
		if runErr != nil {
			return fmt.Errorf("successful response with failed process: %w", runErr)
		}
		return nil
	}
	return response.Error
}
```

Production callers should additionally use a short context deadline, resolve
the helper through an explicit configured path before searching `PATH`, and
never fall back to plaintext storage after a bridge or protocol failure.

## PowerShell probe

The same protocol can be checked from Windows PowerShell:

```powershell
$request = @{
    protocol   = "keyring-winbridge/v1alpha1"
    request_id = "powershell-probe"
    operation  = "probe"
} | ConvertTo-Json -Compress

$request | & .\keyring-winbridge.exe
if ($LASTEXITCODE -ne 0) {
    throw "keyring-winbridge probe failed with exit code $LASTEXITCODE"
}
```

For the complete field definitions and compatibility rules, see the
[`v1alpha1` protocol specification](../docs/protocol/v1alpha1.md).
