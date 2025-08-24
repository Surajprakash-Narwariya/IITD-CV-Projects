# Scalable Backend — Load Balancer · Rate Limiter · In-Memory Cache (Go)

This repo is a compact backend stack built in Go that remains **fast, fair, and resilient under bursts**. It includes:

-   **HTTP reverse-proxy load balancer** with **active health checks** and **round-robin** routing
-   **Per-client rate limiter** using a **token bucket** over a tiny TCP protocol
-   **Thread-safe in-memory cache** exposed via a simple TCP text interface
-   **Application server** that consults the rate limiter, uses the cache (read-through), and serves a JSON API

Everything is runnable locally (no external deps).

---

## 1) What problem it solves (and how)

Modern services fail in 3 common ways under load: **hotspotting** one replica, **unfair** request admission, and **recomputing** the same results. This project composes three primitives to prevent that:

1. **Load Balancer (LB)** spreads requests across replicas and **routes around failures** with active `/health` probes.
2. **Rate Limiter (RL)** does **constant-time admission control** per client (IP or key) using a **token bucket** (rate _R_, burst _B_). Admitted requests proceed; excess get a clean **429**.
3. **In-Memory Cache** turns repeated work into **memory lookups**. The app does **read-through**: `GET`; on miss, compute, then `SET`.

**Why it works**

-   **Health-checked LB** prevents black-holing requests and evens utilization.
-   **Token bucket** provides predictable back-pressure (no queue explosions).
-   **Read-through cache** collapses tail latency for hot keys and reduces CPU load.

---

## 2) Interfaces & behavior (grounded in code)

### Load balancer (HTTP)

-   **Listens** on `:8080`
-   **Backends**: two app replicas at `http://localhost:9001` and `:9002`
-   **Health check**: polls each backend’s `GET /health` periodically; only **healthy** backends receive traffic
-   Transparent reverse-proxy via `httputil.ReverseProxy` (keeps response headers from the app)

### Application server (HTTP)

-   **Listens** on `:9001` / `:9002` (flag: `-port`)
-   **Routes**
    -   `GET /users/<id>` → returns JSON user blob
    -   `GET /health` → `200 OK` when alive (used by LB)
-   **Headers**
    -   `X-Cache-Status: HIT|MISS`
    -   `X-Handled-By: app-server-<port>`
-   **Upstream services it calls**
    -   **Rate limiter** at `localhost:7001` (TCP). Protocol: send `<client-ip>\n`, receive `OK\n` or `ERROR\n`.
    -   **Cache** at `localhost:6379` (TCP). Protocol:
        -   `GET <key>\n` → value or `NIL`
        -   `SET <key> <value>\n` → `OK`
        -   `DEL <key>\n` → `OK`

### Rate limiter (TCP)

-   **Listens** on `:7001`
-   Maintains a **per-IP token bucket** with refill over time and a bounded burst
-   Responds with `OK\n` (admit) or `ERROR\n` (deny) for each check

### Cache (TCP)

-   **Listens** on `:6379`
-   Thread-safe map; newline-delimited commands (`GET/SET/DEL`)
-   No TTL by default (easy to extend)

---

## 3) Build & run

> Requires **Go ≥ 1.20** and a POSIX shell (Linux/macOS).

### Option A — one-shot launcher

```bash
# From the project root
chmod +x run_all.sh
./run_all.sh



```
