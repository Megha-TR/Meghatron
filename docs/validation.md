# Local verification — 2 November 2026

These are local results for the project changes. The CI workflow is supplied as [a template](ci-workflow.yml); publishing credentials lack GitHub workflow permission, so hosted CI is not enabled and no hosted CI result is claimed.

Environment: Go 1.25.0, Node.js 22.23.4, Docker 28.5.0, Linux.

| Check | Result |
| --- | --- |
| `make check` | Formatting, `go vet`, and race-enabled tests passed |
| `make build` | Control plane, worker, and CLI binaries built |
| `make coverage` | 65.3% statement coverage across agent, API, controller, resource, runtime, and scheduler packages |
| `make e2e` | Real etcd/Docker lifecycle test passed |
| `npm run typecheck` | Next route generation and TypeScript passed |
| `npm run lint` | ESLint passed with zero warnings |
| `npm run build` | Production Next.js 16 / React 19 build passed |
| `npm ci --dry-run --offline` | Lockfile/install plan validated |
| `npm audit --omit=dev --audit-level=high` | Zero reported runtime dependency vulnerabilities |
| CLI + `make demo` | Created three available replicas, scaled to one, deleted deployment and owned containers |
| Live Chromium smoke check | Overview/nodes, deployment scaling, pod creation/deletion, Escape dismissal, mobile navigation/layout, and stale data on API failure passed; no browser runtime errors |

The Go suite includes 16 ordinary behavioral/unit tests and one opt-in real-container integration test. Ordinary checks skip the integration test; `make e2e` runs it with a temporary etcd instance and unique Docker ownership labels.

The real-container test verifies three replicas can share container port 80, restart adoption preserves the container count, a stopped container recovers, scale-down removes excess containers, deletion removes all owned containers, duplicate creates fail, and etcd rejects a stale deployment update.

The browser checks used a temporary control plane, worker and production dashboard on separate local ports. Screenshots in this directory's `images/` folder show actual running workloads. The temporary processes, workloads and etcd instances were removed afterward. The browser harness was local verification, not a committed CI browser-test suite.

## Remaining maintenance

The full npm audit reports **16 moderate development-tool advisories** in the ESLint/glob dependency graph. Runtime dependencies audit clean. ESLint 9 is retained because the current React/import/accessibility plugins declare support through ESLint 9; forcing an incompatible major or glob API override is not treated as a verified fix. The prepared CI template checks the production dependency audit. Revisit the development graph as compatible plugin releases become available.

The build/dev scripts explicitly use Webpack. Next.js's Turbopack helper could not bind its local port in this execution environment; the Webpack production build was verified successfully. Type checking and linting remain enabled.

No production availability, throughput, or distributed failover claim is made. See [architecture](architecture.md) for the single-control-plane and trusted-lab boundaries.
