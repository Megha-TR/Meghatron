#!/bin/bash

# Example deployment script for Container Orchestration Lite

set -e

ETCD_ENDPOINT="localhost:2379"
API_SERVER="http://localhost:8080"

echo "=== Container Orchestration Lite - Example Deployment ==="
echo ""

# Check if API server is running
echo "Checking API server health..."
if ! curl -s "$API_SERVER/health" > /dev/null; then
    echo "Error: API server not responding at $API_SERVER"
    echo "Please start the orchestration server first:"
    echo "  ./bin/orchestration-server --port 8080 --etcd $ETCD_ENDPOINT"
    exit 1
fi

echo "API server is healthy"
echo ""

# List existing nodes
echo "=== Current Nodes ==="
curl -s "$API_SERVER/api/v1/nodes" | jq '.[] | {name: .metadata.name, ready: .status.ready, cpu: .capacity.cpu, memory: .capacity.memory}' || echo "No nodes"
echo ""

# Create a test pod
echo "=== Creating Test Pod ==="
POD_JSON=$(cat <<EOF
{
  "metadata": {
    "name": "test-nginx",
    "namespace": "default"
  },
  "spec": {
    "containers": [
      {
        "name": "nginx",
        "image": "nginx:latest",
        "ports": [{"containerPort": 80}]
      }
    ]
  }
}
EOF
)

echo "Creating pod: test-nginx"
RESPONSE=$(curl -s -X POST "$API_SERVER/api/v1/pods" \
  -H "Content-Type: application/json" \
  -d "$POD_JSON")

echo "$RESPONSE" | jq '.'
echo ""

# Create a deployment
echo "=== Creating Deployment ==="
DEPLOYMENT_JSON=$(cat <<EOF
{
  "metadata": {
    "name": "web-app",
    "namespace": "default"
  },
  "spec": {
    "replicas": 3,
    "selector": {"app": "web"},
    "template": {
      "metadata": {"labels": {"app": "web"}},
      "spec": {
        "restartPolicy": "Always",
        "containers": [
          {
            "name": "web",
            "image": "nginx:latest",
            "ports": [{"containerPort": 80}],
            "resources": {
              "requests": {"cpu": "100", "memory": "256"}
            }
          }
        ]
      }
    }
  }
}
EOF
)

echo "Creating deployment: web-app with 3 replicas"
RESPONSE=$(curl -s -X POST "$API_SERVER/api/v1/deployments" \
  -H "Content-Type: application/json" \
  -d "$DEPLOYMENT_JSON")

echo "$RESPONSE" | jq '.'
echo ""

# Wait for reconciliation
echo "=== Waiting for Deployment Reconciliation ==="
sleep 6

# Check deployment status
echo "=== Deployment Status ==="
curl -s "$API_SERVER/api/v1/deployments/default/web-app" | jq '.status'
echo ""

# List all pods
echo "=== All Pods ==="
curl -s "$API_SERVER/api/v1/pods/default" | jq '.[] | {name: .metadata.name, phase: .status.phase, node: .spec.nodeName}' || echo "No pods"
echo ""

echo "=== Example Deployment Complete ==="
