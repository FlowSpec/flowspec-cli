# FlowSpec Security & Offline Mode (PoC)

- Offline-first: CLI runs fully offline; no source code or traces leave your network.
- Inputs: ServiceSpec YAML and exported trace files (OTLP/Jaeger JSON).
- Outputs: Human-readable (Markdown) + JSON reports, written to local disk/stdout.
- Permissions: Read-only access to repos and trace files; no network egress in PoC.
- Determinism: Reports should be deterministic for the same inputs; timestamps are fixed or omitted.
- CI: GitHub Action template uploads reports as artifacts; thresholds configurable.
