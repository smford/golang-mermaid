# Examples & Test Diagrams

This directory provides working examples and test Mermaid diagram files designed to demonstrate and validate the functionality of `golang-mermaid`.

---

## 1. Simple Working Example

Location: [`examples/simple/main.go`](simple/main.go)

A minimal, clean Go application demonstrating:
- Reading a `.mmd` file from disk (or accepting a file argument via CLI).
- Initializing `mermaid.Printer` with `ModeAuto`.
- Registering an SRE observability fallback callback.
- Printing directly to the terminal with iTerm2 inline image support and automatic ASCII fallback.

### Running the Simple Example

```bash
# Run with the default architecture diagram:
go run examples/simple/main.go

# Or provide a custom diagram file:
go run examples/simple/main.go testdata/incident_response.mmd
```

---

## 2. Production CLI Utility (`mermaid-term`)

Location: [`cmd/mermaid-term/main.go`](../cmd/mermaid-term/main.go)

A full-featured command-line utility for developers and SREs with flags for mode selection, dimensions, themes, custom endpoints, and verbose diagnostic logs.

### Installing and Running

```bash
# Build the binary:
go build -o bin/mermaid-term cmd/mermaid-term/main.go

# Render automatically (iTerm2 image when supported, ASCII otherwise):
./bin/mermaid-term testdata/architecture.mmd

# Force ASCII mode:
./bin/mermaid-term -mode=ascii testdata/incident_response.mmd

# Force Unicode box-drawing mode:
./bin/mermaid-term -mode=unicode testdata/sequence_auth.mmd

# Render with verbose SRE diagnostics:
./bin/mermaid-term -v testdata/state_machine.mmd

# Pipe diagram directly from stdin:
cat testdata/database_er.mmd | ./bin/mermaid-term
```

---

## 3. Test Mermaid Diagrams

The [`testdata/`](../testdata/) directory contains 5 realistic Mermaid diagrams modeling real-world Site Reliability Engineering and software architecture scenarios:

| Diagram File | Type | Description |
| :--- | :--- | :--- |
| [`architecture.mmd`](../testdata/architecture.mmd) | **Flowchart** (`graph TD`) | Production cloud architecture featuring CloudFront CDN, ALB, microservices within a Kubernetes VPC, Kafka event bus, Redis cache, PostgreSQL, and external third-party APIs. |
| [`incident_response.mmd`](../testdata/incident_response.mmd) | **Flowchart** (`flowchart TD`) | SRE Incident triage decision tree covering alert ingestion, PagerDuty escalation, severity triage (P1/P2/P3), war rooms, runbook execution, verification, and blameless postmortems. |
| [`sequence_auth.mmd`](../testdata/sequence_auth.mmd) | **Sequence** (`sequenceDiagram`) | Production zero-trust engineer authentication flow: OIDC SSO, WebAuthn MFA, short-lived client certificate issuance via HashiCorp Vault, and Kubernetes API access. |
| [`state_machine.mmd`](../testdata/state_machine.mmd) | **State** (`stateDiagram-v2`) | Kubernetes Pod lifecycle state transitions: `Pending`, `ContainerCreating`, `Running`, `CrashLoopBackOff`, `Terminating`, `Succeeded`, and `Failed`. |
| [`database_er.mmd`](../testdata/database_er.mmd) | **Entity Relationship** (`erDiagram`) | Relational data model connecting Services, Service Level Objectives (SLOs), SLI metrics, and SRE incident assignments. |
