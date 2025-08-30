This folder is for additional vendor OpenAPI specs that you want to use locally.

Guidelines:
- Prefer putting bundled, maintained examples under `examples/specs/`.
- Use `specs/` for local, ad‑hoc, or private specs you don’t plan to publish to npm.
- Point the multi‑host config (`examples/services.example.json`) `specFile` to files in here, e.g. `./specs/your-api.yaml`.

Note: Specs here are not auto‑used by any script; reference them explicitly in configs or CLI flags.

