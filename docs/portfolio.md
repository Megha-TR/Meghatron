# Portfolio and interview notes

Use only claims you can demonstrate. No throughput, uptime, production-readiness, or coverage percentage is claimed here without a corresponding measured result.

## Résumé wording

**Meghatron — Container Orchestration System | Go, etcd, Docker, Next.js, TypeScript**

- Built a Kubernetes-inspired container orchestrator with a REST API, CLI, and dashboard, using etcd to persist desired and observed cluster state.
- Implemented resource-aware best-fit scheduling, deployment reconciliation, replica scaling, and ownership-based container cleanup.
- Added Docker-state reconciliation for agent restart recovery, heartbeat-based worker eligibility, and optimistic concurrency checks to reject stale state updates.
- Validated deployment creation, three-replica execution, restart recovery, scale-down, and deletion with automated tests against real Docker and an isolated etcd instance; prepared a GitHub Actions workflow for repeatable checks.

For a two-bullet version, use the first bullet plus the third, and describe the automated validation in the project link or interview.

## Demonstration

1. Start etcd, the control plane, one worker, and the dashboard using the README.
2. Run `make demo` and show Pending → Running, three available replicas, scale-down, and complete cleanup.
3. Create another deployment, restart its worker process, and show that Docker container IDs are adopted rather than duplicated.
4. Stop one managed container and show recovery on reconciliation.
5. Show a capacity-constrained Pending pod and explain why requests, limits, and actual consumption differ.
6. Open the CI workflow template and explain the distinct unit/behavioral tests and real-container lifecycle test. After enabling the workflow, show its published run result.

## Questions to prepare for

- Why persist deletion intent before removing a container?
- Why does optimistic object concurrency not solve multi-scheduler capacity reservations?
- Why use immutable owner UIDs instead of matching labels to identify deployment pods?
- Why avoid immediately rescheduling pods from an unreachable worker?
- How would watches, queues and backoff improve the polling implementation?
- What separates a Docker Running state from application readiness?
- How would Service networking and shared pod network namespaces change the runtime?

## Next milestones

Application probes and restart backoff; etcd watches/work queues; leader election and fenced node failover; then Service networking and rolling updates. Benchmark only after choosing a target workload and recording the environment and methodology.
