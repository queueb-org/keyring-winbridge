# Security policy

## Supported versions

`keyring-winbridge` is currently released as an alpha project. Security fixes
are provided for the current `0.MINOR` release line on a best-effort basis.

| Version | Supported |
| --- | --- |
| `0.1.x` | Yes, after the first public release |
| Development snapshots and older versions | No |

The supported release line may change when a new pre-1.0 minor version is
published. See the versioning policy in [README.md](README.md).

## Reporting a vulnerability

Do not report a suspected vulnerability in a public issue, discussion, or pull
request. Use **Report a vulnerability** on the repository's **Security** page
to submit it through GitHub Private Vulnerability Reporting.

If private vulnerability reporting is temporarily unavailable, open a public
issue asking the maintainers for a private contact channel. Do not include
vulnerability details, credentials, secret values, diagnostic logs containing
secrets, or reproduction data in that issue.

Include the following in a private report when possible:

- the affected helper and protocol versions;
- the Windows and WSL versions and relevant interoperability configuration;
- a description of the impact and the expected security boundary;
- minimal reproduction steps using synthetic credentials only;
- any suggested mitigation or fix;
- your preferred coordinated-disclosure timeline.

Reports are handled on a best-effort basis. The maintainers will acknowledge a
report when it has been reviewed, keep discussion private while a fix is being
prepared, and coordinate disclosure when practical. This project does not
currently operate a bug bounty program.

## Security boundary

The bridge is a local helper, not a network service. It reads one request from
standard input, writes one response to standard output, and exits. It does not
authenticate callers or add a security boundary between a WSL2 process and the
Windows user context in which the helper is executed.

A process that can execute the helper in that Windows user context can request
credential operations by service and account name. Applications must therefore
restrict who can execute them and must not expose the bridge through a network
listener, shared service, or other untrusted transport.

Secrets are Base64-encoded for transport; Base64 is not encryption. Callers
must protect the helper's standard input and output, avoid logging protocol
messages, invoke the executable directly without a shell, and enforce a short
execution timeout. Secrets must not be passed through command-line arguments
or environment variables.

A transport or protocol failure must be returned to the caller. It must not
silently trigger a fallback to plaintext filesystem storage or another secret
backend.
