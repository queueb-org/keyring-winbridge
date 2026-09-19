# keyring-winbridge protocol v1alpha1

Status: released with `keyring-winbridge v0.1.0`.

Protocol identifier: `keyring-winbridge/v1alpha1`.

## Scope

The protocol lets a Linux process running inside WSL2 perform a small set of
operations against Windows Credential Manager through
`keyring-winbridge.exe`.

The protocol version is independent of both the helper release and the
`common.queueb.org/keyring` release. Clients must verify support for the exact
protocol through `probe`; they must not infer compatibility from executable
versions.

## Process and transport model

Each invocation handles exactly one request:

1. The client starts `keyring-winbridge.exe` directly, without a shell.
2. The client writes one UTF-8 JSON object to standard input and closes it.
3. The helper writes exactly one UTF-8 JSON object to standard output.
4. The helper exits.

Standard output is reserved for the protocol response. Diagnostics may be
written to standard error, but they must never contain service secrets or the
complete request.

Secrets must not be supplied through command-line arguments or environment
variables. Implementations must place a time limit on each invocation and
reject input or output larger than 1 MiB. The decoded `secret_b64` value must
not exceed 2,560 bytes, the Windows limit for a generic credential blob.
Oversized values produce `invalid_argument`.

## Request envelope

Every request has the following fields:

```json
{
  "protocol": "keyring-winbridge/v1alpha1",
  "request_id": "01K5...",
  "operation": "get",
  "service": "example-application",
  "account": "api-token"
}
```

Fields:

- `protocol` is required and must equal the protocol identifier above.
- `request_id` is required, chosen by the client, and treated as an opaque
  string. It is echoed in the response.
- `operation` is required and is one of `probe`, `get`, `set`, or `delete`.
- `service` and `account` are required for credential operations and omitted
  for `probe`.
- `secret_b64` is required only for `set`. It contains standard padded Base64
  so the protocol can preserve arbitrary bytes.

Unknown fields must be ignored. Missing required fields and fields of the
wrong JSON type produce `invalid_request`.

`service` and `account` must be non-empty and must not contain a NUL character.

## Response envelope

A successful response contains `ok: true` and an operation-specific result:

```json
{
  "protocol": "keyring-winbridge/v1alpha1",
  "request_id": "01K5...",
  "ok": true,
  "result": {}
}
```

A failed operation contains `ok: false` and one error object:

```json
{
  "protocol": "keyring-winbridge/v1alpha1",
  "request_id": "01K5...",
  "ok": false,
  "error": {
    "code": "not_found",
    "message": "credential was not found",
    "retryable": false
  }
}
```

Exactly one of `result` and `error` must be present. Error messages are for
diagnostics and must not contain secret values. Clients make decisions using
`code`, not `message`. If an invalid request does not contain a recoverable
`request_id`, the response omits it.

For `unsupported_protocol`, the response echoes the requested protocol and
the error object also contains `supported_protocols`. This minimal error
envelope is the only response that does not imply support for the echoed
protocol.

## Credential identity and backend mapping

Credential operations run in the Windows user context in which the helper was
started. `v1alpha1` stores generic Windows credentials using the mapping
already used by the native Windows backend of `common.queueb.org/keyring`:

- target name: `service + ":" + account`;
- user name: `account`;
- credential blob: the bytes decoded from `secret_b64`;
- persistence: local machine.

This mapping lets a native Windows application and the same application under
WSL2 address the same credential. Windows Credential Manager still scopes the
credential to the Windows user; "local machine" persistence does not make it
available to other users.

The relevant Windows limits and persistence semantics are documented in the
[`CREDENTIALW` structure](https://learn.microsoft.com/en-us/windows/win32/api/wincred/ns-wincred-credentialw).

## Operations

### `probe`

Checks that the executable is a compatible bridge and that the Windows
Credential Manager backend is reachable. It does not access a credential.

Request-specific fields: none.

Result:

```json
{
  "helper_version": "0.1.0",
  "supported_protocols": ["keyring-winbridge/v1alpha1"],
  "backend": "windows-credential-manager"
}
```

The client may cache a successful probe for the lifetime of its storage
instance. The returned helper version is informational only.

### `get`

Reads a credential identified by `service` and `account`.

Result:

```json
{
  "secret_b64": "c2VjcmV0"
}
```

If the credential does not exist, the helper returns `not_found`.

### `set`

Creates or replaces a credential identified by `service` and `account`.

Additional request field:

```json
{
  "secret_b64": "c2VjcmV0"
}
```

Result: an empty object.

### `delete`

Deletes a credential identified by `service` and `account`.

Result:

```json
{
  "deleted": true
}
```

Deleting a missing credential is successful and returns `deleted: false`.
This makes deletion idempotent and maps directly to the boolean deletion API
provided by `common.queueb.org/keyring`.

## Stable error codes

The `v1alpha1` error codes are:

- `invalid_request`: malformed JSON, missing fields, invalid field types, or
  an invalid Base64 value;
- `unsupported_protocol`: the requested protocol is not supported;
- `unsupported_operation`: the requested operation is not supported;
- `invalid_argument`: a syntactically valid value cannot be accepted by the
  backend;
- `not_found`: the requested credential does not exist;
- `access_denied`: Windows denied access to the credential;
- `backend_unavailable`: Windows Credential Manager cannot be reached;
- `backend_failure`: the backend returned an otherwise unclassified failure;
- `timeout`: the operation did not complete within its time limit;
- `internal`: an unexpected helper failure.

`retryable` is advisory. Clients may retry only when both their own policy and
the returned value permit it.

## Process failures

The helper exits with status `0` after an `ok: true` response and status `1`
after an `ok: false` response. Status `2` means the helper could not produce a
valid response. Other exit statuses are reserved. Clients must parse a present
response before interpreting the process exit code.

No response, multiple responses, invalid JSON, an unexpected protocol
identifier, a mismatched `request_id`, or termination by timeout is a transport
failure. A transport failure must not trigger an implicit fallback to
filesystem storage or the Linux kernel keyring.

## Compatibility rules

While this protocol is marked `alpha`, incompatible changes are allowed but
must use a new protocol identifier. Existing `v1alpha1` semantics must not be
silently changed once an executable supporting them has been released.

A pre-1.0 helper `MINOR` release may add or remove supported protocol
identifiers. A `PATCH` release must preserve the protocol set and semantics of
its `0.MINOR` release line.

Additive fields may be introduced without changing the identifier. Clients
and helpers must ignore unknown JSON fields, but they must reject unknown
operations and unsupported protocol identifiers.
