# Scheduling & Collision Avoidance in Networked Systems

> Multithreaded sockets server + many clients to study **ALOHA vs Binary Exponential Backoff (BEB)** for collision avoidance and **FIFO vs Round-Robin (RR)** for flow scheduling. Focus: **throughput stability**, **fairness (JFI)**, and **scalability to 32 clients**.

---

## 1) What problem are we solving?

On a shared medium, **multiple senders contend** for access. Without coordination, overlapping transmissions **collide**, wasting capacity and starving unlucky clients. Even if the link is reliable (TCP), at the **application layer** we still need a policy that decides **when** a client should inject its next burst, and at the receiver we need a **scheduler** to decide **which flow** to serve next. The project implements and evaluates two classic **collision-avoidance** strategies (ALOHA, BEB) and two **server schedulers** (FIFO, RR) to understand their impact on **throughput and fairness** at scale.

---

## 2) How the system works (conceptual, not code)

### A) System model

-   **Server**: a multithreaded process (pthreads) that accepts client connections and **services application-layer bursts**. It maintains per-client queues and exposes **two scheduling policies**:
    -   **FIFO**: serve flows in arrival order until each finishes its burst.
    -   **Round-Robin**: cycle across active flows; each gets a **quantum** of bytes/packets before moving to the next.
-   **Clients**: independent senders that **decide when to transmit** their next application burst based on the chosen **collision-avoidance protocol**.

> Think of the server as a _shared channel endpoint_ and the clients as _stations_ trying to “speak” without talking over each other.

### B) Collision-avoidance protocols at the client

-   **ALOHA (pure, unslotted)**  
    Each client attempts a send **as soon as it has data**. If it doesn’t receive an ACK within a timeout window (interpreted as a collision or loss), it waits a **random backoff** from a **fixed window** and retries. This is simple but can exhibit **instability** when loads are high (more overlap → more collisions).

-   **Binary Exponential Backoff (BEB)**  
    Upon each failed attempt, the client **doubles** its contention window (e.g., CW ← min(2·CW, CWmax)) and samples a random wait in `[0, CW)`. A success **resets** the window toward **CWmin**. BEB adapts aggressiveness to contention: under heavy load, clients **back off more**, reducing overlap and stabilizing aggregate throughput.

**ACK/timeout model.** The server ACKs an application burst after it is admitted and scheduled. Lack of ACK within **RTO** implies a collision/queueing timeout, triggering the client’s backoff rule (ALOHA or BEB). **Karn-style** avoidance is used: RTT samples from retransmitted bursts are not used to update RTO.

### C) Server-side scheduling (who gets service next?)

-   **FIFO** favors the **head-of-line** flow; long bursts can **block** shorter ones, causing **fairness issues** and bursty tail latencies.
-   **Round-Robin (RR)** **time-slices** service across active flows. Small quanta yield **smooth sharing** (lower jitter, better fairness) at the cost of more context switches; large quanta drift toward FIFO.

### D) Metrics & methodology

-   **Throughput / Goodput** per client and system-wide.
-   **Fairness via Jain’s Fairness Index (JFI)**:
    \[
    \text{JFI} = \frac{\left(\sum*{i=1}^{n} x_i\right)^2}{n \cdot \sum*{i=1}^{n} x_i^2},\quad 0 < \text{JFI} \le 1
    \]
    where \(x_i\) is client \(i\)’s achieved throughput; **higher is fairer**.
-   **Latency / Completion time** distributions; **collision rate** (implied by timeouts/retries).

**Headline result.** With BEB on the clients and RR at the server, we observed a **~29% improvement in JFI** over the ALOHA+FIFO baseline while sustaining **stable throughput** up to **32 concurrent clients** (hardware-dependent).

---

## 3) Why these designs work (intuition)

-   **ALOHA** is aggressive and simple → good when lightly loaded, but it **self-amplifies collisions** as more clients join.
-   **BEB** responds to contention: after a failure, each client **widens** its random wait window. This **decorrelates** retransmissions, reducing repeated overlaps and stabilizing the system.
-   **FIFO** maximizes single-flow efficiency but can **starve** short flows behind long ones (poor short-term fairness).
-   **RR** enforces **temporal fairness**; everyone progresses a bit each round, which **raises JFI** and smooths latency tails.

---

## 4) How to run (generic)

> Requires a POSIX system with a C++17 toolchain and Python 3 for plotting. The project includes a server binary, a client binary, and a small harness for parameter sweeps. Names/paths may differ in your checkout; use the provided `--help` on each binary.

### Build

```bash
# From project root
make            # or your usual build command
```
