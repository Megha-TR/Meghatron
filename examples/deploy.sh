#!/usr/bin/env bash
# Requires a running API and worker; creates only the unique deployment below.
set -euo pipefail
cd "$(dirname "$0")/.."
api="${MEGHATRON_SERVER:-http://127.0.0.1:8080}"
name="demo-$(date +%s)-$$"
path="$api/api/v1/deployments/default/$name"
for cmd in curl jq; do command -v "$cmd" >/dev/null || { echo "Install $cmd first" >&2; exit 1; }; done
curl -fsS "$api/ready" >/dev/null
if ! curl -fsS "$api/api/v1/nodes" | jq -e 'any(.[]; .status.ready)' >/dev/null; then echo 'Start a node agent first' >&2; exit 1; fi
cleanup() { curl -fsS -X DELETE "$path" >/dev/null 2>&1 || true; }
trap cleanup EXIT
body="$(jq --arg name "$name" '.metadata.name=$name' examples/web-deployment.json)"
curl -fsS -X POST "$api/api/v1/deployments" -H 'Content-Type: application/json' -d "$body" | jq '{name:.metadata.name, desired:.spec.replicas}'
wait_replicas() {
  for _ in $(seq 1 120); do
    if curl -fsS "$path" | jq -e --argjson n "$1" '.status.availableReplicas == $n and .status.replicas == $n' >/dev/null; then return; fi
    sleep 1
  done
  echo 'Timed out; inspect the Pods page for scheduling or container errors.' >&2; exit 1
}
wait_replicas 3
echo 'Three nginx replicas are ready. Scaling to one…'
curl -fsS -X PATCH "$path/scale" -H 'Content-Type: application/json' -d '{"replicas":1}' >/dev/null
wait_replicas 1
echo 'One replica is ready. Deleting the deployment…'
curl -fsS -X DELETE "$path" >/dev/null
for _ in $(seq 1 60); do
  code="$(curl -sS -o /dev/null -w '%{http_code}' "$path")"
  if [[ "$code" == 404 ]]; then trap - EXIT; echo 'Deployment and owned containers cleaned up.'; exit 0; fi
  sleep 1
done
echo 'Cleanup is still pending; check the worker agent.' >&2
exit 1
