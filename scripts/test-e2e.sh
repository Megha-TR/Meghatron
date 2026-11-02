#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
name="meghatron-test-etcd-$$"
cleanup() { docker rm -f "$name" >/dev/null 2>&1 || true; }
trap cleanup EXIT
# Random loopback port isolates this run from any existing etcd instance.
docker run -d --name "$name" -p 127.0.0.1::2379 \
  quay.io/coreos/etcd:v3.5.17 \
  /usr/local/bin/etcd --listen-client-urls=http://0.0.0.0:2379 \
  --advertise-client-urls=http://127.0.0.1:2379 >/dev/null
port="$(docker port "$name" 2379/tcp | awk -F: '{print $NF}')"
ready=false
for _ in $(seq 1 60); do
  if curl -fsS "http://127.0.0.1:$port/health" >/dev/null; then ready=true; break; fi
  sleep 1
done
if [[ "$ready" != true ]]; then docker logs "$name"; exit 1; fi
MEGHATRON_TEST_ETCD="127.0.0.1:$port" go test -race -count=1 -timeout=5m -v ./pkg/integration
