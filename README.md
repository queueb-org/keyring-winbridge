# keyring-winbridge

`keyring-winbridge` is a small Windows helper that gives applications running
inside WSL2 access to Windows Credential Manager.

The helper is distributed independently from the
`common.queueb.org/keyring` Go library. Applications embed the library, while
users who want Windows Credential Manager integration install
`keyring-winbridge.exe` separately.

The library communicates with the helper over a versioned stdin/stdout
protocol. It does not download, install, or update the helper.

## Versioning before 1.0

Until the first stable release, `keyring-winbridge` uses versions in the form
`0.MINOR.PATCH`. During this period, `MINOR` acts as the compatibility boundary:

- a new `MINOR` version may contain changes incompatible with previous minor
  versions; backward compatibility is not guaranteed across `0.MINOR`
  releases;
- a new `PATCH` version remains backward compatible within its minor release
  line;
- consumers should check protocol compatibility instead of inferring it from
  the application version.

Starting with `1.0.0`, releases follow Semantic Versioning in the conventional
way.

The executable version and wire-protocol version are independent. The initial
protocol is specified in [docs/protocol/v1alpha1.md](docs/protocol/v1alpha1.md).
Runnable protocol examples are collected in
[examples/README.md](examples/README.md).

## Supported platform

The supported binary target for `v0.1.0` is `windows/amd64`. Release binaries
use the `GOAMD64=v1` baseline for compatibility with 64-bit Windows systems.

The intended application topology is:

1. a Linux application runs inside WSL2;
2. it starts `keyring-winbridge.exe` directly through Windows executable
   interoperability;
3. the helper runs in the intended Windows user context and accesses that
   user's Windows Credential Manager.

The executable can also be invoked directly on Windows for installation
checks, troubleshooting, and protocol diagnostics. WSL1, Windows on ARM,
Wine, and environments without Windows executable interoperability are not
part of the tested `v0.1.0` support target. A non-Windows build uses an
unsupported backend and cannot access Windows Credential Manager.

## Installation

The recommended installation for users is the prebuilt
`keyring-winbridge.exe` published with a GitHub Release. Keep the executable in
a location available to the Windows user and executable from WSL2.

The Go toolchain is also a supported way to obtain the executable. On Windows,
install a specific release with:

```powershell
go install queueb.org/keyring-winbridge@v0.1.0
```

Using `@latest` is also valid:

```powershell
go install queueb.org/keyring-winbridge@latest
```

From WSL2, explicitly request a Windows executable:

```bash
GOOS=windows GOARCH=amd64 GOAMD64=v1 \
  go install queueb.org/keyring-winbridge@v0.1.0
```

For a cross-install with the default Go configuration, the executable is
written to:

```text
$(go env GOPATH)/bin/windows_amd64/keyring-winbridge.exe
```

Do not set `GOBIN` for this cross-install: the Go toolchain does not install
cross-compiled binaries when `GOBIN` is set. Running `go install` inside WSL2
without `GOOS=windows` builds an unsupported Linux executable rather than the
Windows bridge.

Pinning a tag is recommended for reproducible installation. Before `1.0.0`, a
new `0.MINOR` release may intentionally be incompatible with earlier minor
releases, so `@latest` may move to a new compatibility line.

## Development

The scripts in `hack/` use `hack/.build-env.sh`, which sets `windows/amd64`
with the `GOAMD64=v1` baseline as the target:

```bash
./hack/test.sh
./hack/lint.sh
./hack/build.sh
```

These commands do not modify Windows Credential Manager. The opt-in integration
test creates a uniquely named test credential, verifies the complete CRUD
cycle, and removes the credential during cleanup:

```bash
./hack/integration-test.sh
```

Run the integration test on Windows or from WSL with Windows executable
interop enabled.

## Status

`v0.1.0` is the first public alpha release. It provides the released
`keyring-winbridge/v1alpha1` protocol for Windows Credential Manager access
from WSL2. The protocol remains alpha, but its published `v1alpha1` semantics
are frozen according to the compatibility rules in its specification.
