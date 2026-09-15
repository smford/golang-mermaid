# Examples & Test Diagrams

This directory provides working examples, production CLI usage patterns, and test Mermaid diagrams designed to demonstrate and validate the full functionality of `golang-mermaid`.

---

## 1. Simple Working Example

Location: [`examples/simple/main.go`](simple/main.go)

A complete Go application demonstrating senior developer and SRE practices:
- Reading either standard `.mmd` diagram files or Markdown documents (`.md`).
- Multi-protocol graphics auto-detection (Kitty, iTerm2, Sixel) with graceful ASCII fallback.
- Content-addressed disk and memory caching (`WithCache(true)`).
- In-band PTY capability probing for remote SSH sessions (`WithTerminalProbe(true)`).
- SRE observability fallback callbacks (`WithOnFallback`).
- Built-in performance and operational metrics recording (`WithTelemetry(stats)`).
- Printing an execution telemetry summary on exit.

### Running the Simple Example

```bash
# Run with the default cloud architecture diagram:
go run examples/simple/main.go

# Run with an incident response flowchart:
go run examples/simple/main.go testdata/incident_response.mmd

# Run with an incident response Markdown runbook (renders embedded diagrams inline):
go run examples/simple/main.go testdata/runbook.md
```

---

## 2. Production CLI Utility (`mermaid-term`)

Location: [`cmd/mermaid-term/main.go`](../cmd/mermaid-term/main.go)

A full-featured command-line utility for engineers and SREs.

### Installing & Building

```bash
# Build binary into bin/
go build -o bin/mermaid-term cmd/mermaid-term/main.go
```

### CLI Flag Reference

| Flag | Type | Description | Default |
| :--- | :--- | :--- | :--- |
| `-file` | string | Path to `.mmd`, `.mermaid`, or `.md` file (or provide as first positional argument) | `""` (stdin) |
| `-mode` | string | Rendering mode: `auto`, `image`, `ascii`, or `unicode` | `auto` |
| `-protocol` | string | Terminal graphics protocol: `auto`, `kitty`, `iterm2`, `sixel`, `none` | `auto` |
| `-probe` | bool | Query terminal capabilities in-band via PTY escapes (`\033[c`) for SSH | `false` |
| `-interactive` | bool | Launch interactive 2D pan & zoom terminal pager for large diagrams | `false` |
| `-cache` | bool | Enable content-addressed disk caching | `true` |
| `-cache-dir` | string | Custom disk cache directory | `~/.cache/golang-mermaid` |
| `-cache-ttl` | duration | Time-to-live for cached diagram renders (e.g. `24h`, `30m`) | `24h` |
| `-clear-cache` | bool | Clear cached diagrams and exit | `false` |
| `-offline` | bool | Air-gapped mode: disable all remote HTTP calls (uses local `mmdc` or text) | `false` |
| `-markdown` | bool | Parse Markdown document and render embedded ` ```mermaid ` blocks inline | `false` (auto if `.md`) |
| `-metrics` | bool | Export Prometheus-compatible telemetry metrics to `stderr` on exit | `false` |
| `-width` | string | Image display width (e.g. `80%`, `800px`, `60cell`) | `auto` |
| `-height` | string | Image display height (e.g. `400px`, `30cell`) | `auto` |
| `-scale` | float | Rasterization scale factor for HiDPI/Retina screens (`1.0`, `2.0`, `3.0`) | `1.0` |
| `-theme` | string | Text/ASCII diagram theme (`default`, `slate`, `blueprint`, `neon`, `amber`, `monokai`)| `default` |
| `-frame` | bool | Wrap text/ASCII diagram in an executive card border | `false` |
| `-title` | string | Header title for executive card border | `""` |
| `-columns` | int | Column width override for text diagram layout (0 for auto-detection) | `0` |
| `-v` | bool | Verbose SRE diagnostic logs (mode, protocol, duration, fallback reason) | `false` |

### CLI Usage Examples

```bash
# 1. Automatic protocol detection (Kitty, iTerm2, Sixel, or ASCII):
./bin/mermaid-term testdata/architecture.mmd

# 2. Specific graphics protocol selection:
./bin/mermaid-term -protocol=kitty testdata/sequence_auth.mmd
./bin/mermaid-term -protocol=sixel testdata/state_machine.mmd

# 3. Interactive 2D pan & zoom pager (navigate with arrow keys or hjkl, q to exit):
./bin/mermaid-term -interactive testdata/architecture.mmd

# 4. In-band PTY probing for SSH sessions:
./bin/mermaid-term -probe testdata/architecture.mmd

# 5. Render a full Markdown runbook replacing ```mermaid blocks inline:
./bin/mermaid-term testdata/runbook.md

# 6. Air-gapped / offline mode (zero remote HTTP network requests):
./bin/mermaid-term -offline testdata/architecture.mmd

# 7. SRE Prometheus telemetry metrics exported to stderr:
./bin/mermaid-term -metrics testdata/architecture.mmd

# 8. Force ASCII mode with dark slate styling and card framing:
./bin/mermaid-term -mode=ascii -frame -title="Incident Triage Flow" -theme=slate testdata/incident_response.mmd

# 9. Clear cache:
./bin/mermaid-term -clear-cache

# 10. Pipe diagram directly from stdin:
cat testdata/database_er.mmd | ./bin/mermaid-term
```

---

## 3. Test Mermaid Diagrams & Runbooks

The [`testdata/`](../testdata/) directory contains realistic diagrams and runbooks modeling real-world Site Reliability Engineering scenarios:

| File | Type | Description |
| :--- | :--- | :--- |
| [`architecture.mmd`](../testdata/architecture.mmd) | **Flowchart** (`graph TD`) | Production cloud architecture: CloudFront CDN, ALB, microservices within a Kubernetes VPC, Kafka event bus, Redis cache, PostgreSQL, and third-party APIs. |
| [`incident_response.mmd`](../testdata/incident_response.mmd) | **Flowchart** (`flowchart TD`) | SRE incident triage decision tree covering alert ingestion, PagerDuty escalation, severity triage (P1/P2/P3), war rooms, runbook execution, verification, and blameless postmortems. |
| [`sequence_auth.mmd`](../testdata/sequence_auth.mmd) | **Sequence** (`sequenceDiagram`) | Production zero-trust engineer authentication flow: OIDC SSO, WebAuthn MFA, short-lived client certificate issuance via HashiCorp Vault, and Kubernetes API access. |
| [`state_machine.mmd`](../testdata/state_machine.mmd) | **State** (`stateDiagram-v2`) | Kubernetes Pod lifecycle state transitions: `Pending`, `ContainerCreating`, `Running`, `CrashLoopBackOff`, `Terminating`, `Succeeded`, and `Failed`. |
| [`database_er.mmd`](../testdata/database_er.mmd) | **Entity Relationship** (`erDiagram`) | Relational data model connecting Services, Service Level Objectives (SLOs), SLI metrics, and SRE incident assignments. |
| [`runbook.md`](../testdata/runbook.md) | **Markdown Runbook** (`.md`) | Incident response operations document embedding Markdown text, bash commands, and live Mermaid diagrams rendered inline. |
