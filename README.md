# Meghatron — Container Orchestration

A lightweight Kubernetes-like container orchestration system built from scratch in Go, with a full-stack web dashboard. This project demonstrates distributed systems concepts including scheduling, resource management, controller reconciliation, and container lifecycle management.

## What It Does

You have apps (packaged as Docker containers) and servers. This system is the traffic controller that:

- Decides **which server runs which container** (bin-packing scheduler)
- **Pulls images and starts containers** automatically via Docker
- Keeps the **right number of replicas** running (deployment controller)
- Lets you manage everything from a **CLI, REST API, or web dashboard**

It is a simplified version of Kubernetes built from scratch to demonstrate how orchestration systems work internally.

---

## Architecture

```
┌─────────────────────────────────────────┐
│         Web Dashboard (Next.js)         │
│  • Overview, Nodes, Pods, Deployments   │
│  • Create / delete resources via UI     │
│  • Auto-refreshes every 5 seconds       │
└─────────────────────────────────────────┘
              ↓ (HTTP proxy :3000 → :8080)
┌─────────────────────────────────────────┐
│      Orchestration Server (Master)      │
├─────────────────────────────────────────┤
│  • REST API Server (chi router)         │
│  • Scheduler (bin-packing algorithm)    │
│  • Deployment Controller                │
│  • etcd integration for state storage   │
└─────────────────────────────────────────┘
              ↓ (HTTP / etcd)
┌─────────────────────────────────────────┐
│      Node Agent (Worker Node)           │
├─────────────────────────────────────────┤
│  • Registers with master                │
│  • Polls for scheduled pods (5s loop)   │
│  • Pulls Docker images                  │
│  • Creates and starts containers        │
│  • Sends heartbeats every 10s           │
└─────────────────────────────────────────┘
```

### Request Flow (pod create)

```
orc pod create nginx nginx:latest
        │
        ▼
POST /api/v1/pods  (orchestration-server :8080)
        │
        ▼
Scheduler: finds best node via bin-packing
        │
        ▼
Saves pod with NodeName=worker-1 to etcd
        │
        ▼
Node agent polls etcd, sees pod assigned to it
        │
        ▼
Pulls nginx:latest → docker create → docker start
        │
        ▼
Container running, pod phase updated to Running
        │
        ▼
Dashboard auto-refreshes and shows Running status
```

---

## Project Structure

```
project/
├── cmd/
│   ├── orchestration-server/    # Master API server entry point
│   ├── node-agent/              # Worker node agent entry point
│   └── cli/                     # orc CLI tool entry point
├── pkg/
│   ├── api/                     # REST API handlers (chi router)
│   ├── scheduler/               # Bin-packing scheduling algorithm
│   ├── controller/              # Deployment reconciliation loop
│   ├── storage/                 # etcd storage layer (CRUD for all resources)
│   ├── runtime/                 # Docker container runtime (CLI-based)
│   └── types/                   # Core data types (Pod, Node, Deployment, etc.)
├── web/                         # Next.js + Tailwind web dashboard
│   ├── src/app/                 # Pages: overview, nodes, pods, deployments
│   └── src/components/          # Sidebar, StatusBadge
├── bin/                         # Compiled binaries (after build, gitignored)
├── go.mod
└── README.md
```

---

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend language | Go 1.21+ |
| HTTP router | go-chi/chi v5 |
| State store | etcd v3.5 |
| Container runtime | Docker (via CLI) |
| CLI framework | urfave/cli v2 |
| Frontend | Next.js 14 + React 18 |
| Styling | Tailwind CSS |
| ID generation | google/uuid |

---

## Requirements

- **Go 1.21+** — `go version`
- **Node.js 18+** — `node --version`
- **Docker Desktop** — must be open and running
- **etcd** — runs inside Docker, no separate install needed

---

## Full Setup — Run Everything From Scratch

### Step 1 — Open Docker Desktop

On macOS: press Cmd+Space, type Docker, hit Enter. Wait for the whale icon in the menu bar to stop animating.

### Step 2 — Start etcd

**First time:**
```bash
docker run -d --name etcd \
  -p 2379:2379 \
  -e ETCD_LISTEN_CLIENT_URLS=http://0.0.0.0:2379 \
  -e ETCD_ADVERTISE_CLIENT_URLS=http://127.0.0.1:2379 \
  quay.io/coreos/etcd:v3.5.0
```

**After first time:**
```bash
docker start etcd
```

### Step 3 — Build the Go binaries

```bash
cd /path/to/project
go mod download
go build -o bin/orchestration-server ./cmd/orchestration-server
go build -o bin/node-agent ./cmd/node-agent
go build -o bin/orc ./cmd/cli
```

### Step 4 — Install dashboard dependencies

```bash
cd web && npm install && cd ..
```

### Step 5 — Start everything (4 terminals)

**Terminal 1 — orchestration server:**
```bash
./bin/orchestration-server --port 8080 --etcd localhost:2379
```

**Terminal 2 — node agent:**
```bash
./bin/node-agent --node-name worker-1 --etcd localhost:2379
```

**Terminal 3 — web dashboard:**
```bash
cd web && npm run dev
```

**Terminal 4 — CLI:**
```bash
./bin/orc node list
```

---

## How to Test It's Working

### Web Dashboard

Open **http://localhost:3000** — you should see the Meghatron dashboard with live node/pod/deployment data.

### CLI

```bash
# List nodes
./bin/orc node list

# Create a pod
./bin/orc pod create nginx nginx:latest

# Check pod is running
./bin/orc pod list

# Confirm Docker ran it
docker ps

# Create a deployment with 3 replicas
./bin/orc deployment create web-app nginx:latest 3

# Delete a pod
./bin/orc pod delete nginx
```

### REST API

```bash
# Health check
curl http://localhost:8080/health

# List nodes
curl http://localhost:8080/api/v1/nodes

# Create a pod
curl -X POST http://localhost:8080/api/v1/pods \
  -H "Content-Type: application/json" \
  -d '{
    "metadata": {"name": "my-pod", "namespace": "default"},
    "spec": {
      "containers": [{"name": "app", "image": "nginx:latest"}]
    }
  }'

# Delete a pod
curl -X DELETE http://localhost:8080/api/v1/pods/default/my-pod
```

---

## Troubleshooting

**etcd connection refused:**
```bash
docker start etcd
```

**Docker binary not found (node agent error):**
Docker Desktop must be open. The runtime auto-checks these paths:
`docker` → `/usr/local/bin/docker` → `/usr/bin/docker` → `/Applications/Docker.app/Contents/Resources/bin/docker`

**Pod stuck in Pending:**
Node agent must be running. Check its terminal for errors.

**Container name conflict on restart:**
```bash
docker rm -f default-nginx-nginx
```
Node agent recreates it within 5 seconds.

**Dashboard shows 0s:**
```bash
curl http://localhost:8080/health   # server must be running
```

---

## Core Components

### Scheduler (`pkg/scheduler/`)
Bin-packing algorithm — filters nodes by available CPU/memory, picks the one with minimum waste.

### Deployment Controller (`pkg/controller/`)
Reconciliation loop every 5 seconds — compares desired vs actual replicas, creates pods to fill the gap.

### Container Runtime (`pkg/runtime/`)
Wraps Docker CLI via `os/exec` — auto-detects Docker binary, pulls images on demand, scopes container names as `{namespace}-{pod}-{container}`.

### Storage (`pkg/storage/`)
etcd-backed with hierarchical keys: `/pods/{namespace}/{name}`, `/nodes/{name}`, `/deployments/{namespace}/{name}`.

### Web Dashboard (`web/`)
Next.js 14 + Tailwind — 4 pages (Overview, Nodes, Pods, Deployments), create/delete from UI, auto-refreshes every 5s, proxies API via Next.js rewrites.

---

## Known Limitations

- No scale-down — deployment controller scales up only
- Pod deletion does not stop the Docker container
- No health checks (liveness/readiness probes)
- No service discovery or DNS
- No authentication on the API
- No rolling updates

---

## Security Notice

This is a learning/portfolio project. Do not use in production without authentication, TLS, resource isolation, and audit logging.

---

## License

MIT License
