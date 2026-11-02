# Design and failure behavior

## Desired state and reconciliation

The API validates and persists intent rather than requiring capacity at request time. A pod created before its worker registers remains Pending. The scheduler retries unassigned pods each second, including deployment-created pods and pods outside the default namespace.

The deployment controller owns pods by deployment UID, not by label coincidence. It creates missing replicas, marks excess or terminal pods for deletion, and counts availability only when every container is observed running. Scale updates and deletes use optimistic writes, so a stale controller snapshot cannot overwrite a newer desired replica count or deletion request. The current template cannot be updated through the API; `updatedReplicas` therefore describes replicas of the single supported template.

## Resource scheduling

The scheduler calculates free resources from node allocatable capacity minus all assigned pod requests, including terminating pods awaiting cleanup. CPU/memory units are parsed into millicores/bytes. It filters unhealthy/expired workers and host-port conflicts, then minimizes normalized remaining CPU plus memory. Sorting nodes by name makes ties deterministic.

Assignment is serialized inside one scheduler process. Every record write also checks etcd's modification revision. This protects object consistency; it is not a distributed capacity transaction. Multiple active schedulers are out of scope until leader election or transactional reservations are added.

## Container ownership and restart

Containers carry `meghatron.managed`, `meghatron.node`, `meghatron.podUID`, `meghatron.container`, namespace, and pod-name labels. Their Docker name incorporates the immutable pod UID. Every reconciliation lists labeled containers for that worker and compares actual state to etcd; no in-memory registry is needed for recovery.

Existing containers are adopted after a process restart. Missing containers are created. Exited containers restart under `Always`, restart only after nonzero exit under `OnFailure`, and remain stopped under `Never`. A failed sidecar does not cause an already running sibling to be recreated. Docker restart policy is disabled so the agent owns these decisions.

Container creation/start errors remain Pending with an explanatory message and retry each cycle. There is currently no exponential backoff or image-pull concurrency; a slow pull can delay other pods on a worker. Heartbeats run independently from pulls.

## Deletion and consistency

Public deletion writes a timestamp instead of erasing the record. The worker removes all labeled containers whose pod is missing or terminating; only after successful cleanup does it erase terminating pod records. The scheduler may directly erase a terminating unassigned pod because no worker has run it. Deployment deletion marks all owned pods and waits for their records to disappear.

etcd writes compare the caller's resource version with the current key revision. Version zero is create-only. Duplicate names and stale writes fail with a conflict, including stale status writes after deletion. List responses include observed revisions, and malformed persisted records cause an error instead of silently disappearing from the list. A controller conflict is retried on its next pass.

## Worker failure and shutdown

Workers send a heartbeat every ten seconds. After thirty seconds without a fresh heartbeat they are excluded from new scheduling and reported NotReady. Existing pods are not automatically rescheduled: without fencing, a partitioned worker could still run its containers. That deliberate limitation avoids pretending to offer safe high availability.

The control plane stops HTTP gracefully and waits for reconciliation loops to exit before closing storage. The worker cancels its Docker operations and waits for its heartbeat loop on shutdown. Agent shutdown intentionally leaves workloads running for adoption on restart.

## Boundaries

The local API is unauthenticated, and workers communicate directly with trusted etcd. A production architecture would require an authenticated worker API, TLS, authorization, resource isolation, auditing, leader election, and fencing. Service and pod networking, application probes, rolling updates, and persistent volumes remain separate future work.
