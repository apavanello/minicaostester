# Chaos & Load Target MVP

A lightweight, zero-dependency Go application designed to serve as a reliable target for load testing and chaos engineering experiments.

It simulates architectural bottlenecks by manipulating CPU utilization, network latency (I/O), and error cascades from downstream integrations, entirely in memory without relying on databases or disk persistence.

## Core Features

*   **Zero Dependencies:** Built exclusively with Go's standard `net/http` library. Ships as a `scratch` Docker image under 10MB.
*   **CPU Burner:** Executes mathematical loops to intentionally stress processors, simulating complex organic workloads.
*   **Synthetic Latency:** Introduces configurable random `sleep` intervals to mimic slow I/O (e.g., degraded databases or APIs).
*   **Service Chaining (BFF Mode):** Acts as an aggregator, invoking multiple external URLs concurrently via Goroutines to simulate network dependencies.
*   **Circuit Breaker:** Implements depth-based request limiting via the `X-Call-Depth` header to prevent infinite recursive loops within the cluster.

## Configuration (Environment Variables)

The application's behavior is entirely controlled via environment variables:

| Variable | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `PORT` | `int` | `8080` | Local HTTP server port. |
| `MIN_DELAY_MS` | `int` | `10` | Minimum wait time in Standalone mode. |
| `MAX_DELAY_MS` | `int` | `100` | Maximum wait time for variance. |
| `BURN_CPU` | `bool` | `false` | Enables synthetic CPU stress. |
| `CPU_COMPLEXITY` | `int` | `50000` | Number of mathematical iterations per request (Burn mode). |
| `EXTERNAL_SERVICES`| `string`| `""` | Comma-separated list of URLs for Service Chaining. |
| `MAX_CALL_DEPTH` | `int` | `5` | Maximum allowed request chaining depth. |
| `REQUEST_TIMEOUT` | `int` | `5` | Timeout (in seconds) for external HTTP calls. |

## Running Locally

### Using Go Toolchain
```bash
make build
make run

# Or via CLI with custom ENV:
PORT=9000 BURN_CPU=true go run main.go
```

Test the endpoint:
```bash
curl http://localhost:8080/
```

### Using Docker
```bash
docker build -t chaos-target:local .
docker run --rm -p 8080:8080 -e BURN_CPU=true -e MAX_DELAY_MS=500 chaos-target:local
```

## Kubernetes Deployment (GitOps)

The repository provides a Helm chart located at `deploy/helm`, tailored for GitOps workflows (e.g., ArgoCD).

1. Adjust the chaos thresholds in `deploy/helm/values.yaml` under the `chaos:` key.
2. Install via Helm:
```bash
helm upgrade --install chaos-target ./deploy/helm -n my-chaos-namespace
```

## Architecture & Documentation

Refer to the `./docs` directory for architectural decisions and models:
*   [C4 Components](./docs/arquitetura-componentes.md)
*   [Internal Flowchart](./docs/fluxos.md)
*   [JSON Payload Models](./docs/modelo-dados.md)
*   [Sequence Diagrams](./docs/diagramas-sequencia.md)
