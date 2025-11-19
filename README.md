# goeth

`goeth` is a small, script-friendly command-line utility that inspects network
interfaces, reports assigned addresses, and validates JSON configuration files
before applying them. It is designed for operators who need a trustworthy,
repeatable way to audit interface state or dry-run configuration changes without
shelling out to multiple platform-specific tools.

## Project goals & capabilities

* **Network interface inventory** – enumerate all detected interfaces, their
  MTU, and hardware (MAC) address, sorted for easy scanning.
* **Address inspection** – show every IPv4/IPv6 address assigned to a specific
  interface so you can verify live state or diagnose configuration drift.
* **Configuration application** – load a JSON document that declares the target
  interface plus a list of addresses and apply it via a validated executor that
  adds/removes the prefixes on the live interface (a `--dry-run` flag keeps the
  previous console-only behavior).

These commands share a consistent Cobra-based interface and emit human-readable
output that can also be parsed by higher-level orchestration tools.

## Installation

Prerequisites:

* Go **1.22** or newer (per `go.mod`).
* A writable `GOBIN`/`GOPATH/bin` on your `PATH` if you plan to install the
  binary globally.

Install options:

1. **From source (recommended for contributors)**
   ```bash
   git clone https://github.com/user/goeth.git
   cd goeth
   go build -o bin/goeth ./cmd/app
   # or install globally
   go install ./cmd/app
   ```
2. **Go install via module path** (no clone required)
   ```bash
   go install github.com/user/goeth/cmd/app@latest
   ```

After either method the `goeth` binary is available in `bin/` or your `GOBIN`.

## Usage examples

List the interfaces detected on the current machine:

```bash
goeth interfaces
```

Inspect all addresses assigned to `eth0`:

```bash
goeth addresses --interface eth0
```

Continuously watch interfaces (optionally filtered) and emit a log whenever
their properties or addresses change:

```bash
goeth monitor --interval 10s --interface eth0
```

Apply a configuration defined in JSON (validated before execution):

```bash
cat <<'JSON' > cfg.json
{
  "interface": "eth0",
  "addresses": [
    "192.0.2.10/24",
    "2001:db8::10/64"
  ]
}
JSON

goeth apply-config --file cfg.json
# or review the operations without touching the network
goeth apply-config --file cfg.json --dry-run
```

The sample configuration uses the `interface` field to choose the target
interface and `addresses` to list each prefix that should be attached. By
default `goeth apply-config` now configures the OS directly (via Netlink) to
match those addresses; pass `--dry-run` to fall back to the console executor if
you only want to review the proposed changes.

## Dependency notes

* Runtime functionality relies on the Go standard library (`net`, `os`, etc.)
  plus [spf13/cobra](https://github.com/spf13/cobra) for the CLI surface.
* Development tooling references `golang.org/x/*` helpers (`staticcheck`,
  `govulncheck`, telemetry) – install them via `go install` if you want to run
  the optional Make targets locally.
* `go install` (as shown above) automatically resolves and caches all needed
  modules; no extra package manager is required.

## Contributing & testing

1. Fork or branch from `main`, then add your changes.
2. Keep documentation and sample commands in sync when altering CLI behavior.
3. Before opening a pull request, run the same checks enforced by CI:
   ```bash
   make lint
   make test
   make build
   ```
   (All targets live in the root `Makefile` and assume standard Go tooling.)
4. Include focused unit tests alongside code changes (`*_test.go` files live
   next to the code they exercise) and update fixtures under `test/` when
   relevant.
5. Open a PR with context on Motivation, Design, Tests, and Risks as described
   in `AGENTS.md`.

Issues and feature requests are welcome. Please reference any relevant Go
version or platform details when reporting bugs so maintainers can reproduce the
problem quickly.
