# Reliable File Transfer over UDP — Sliding Window + RTT-Based Timeout + TCP Reno & CUBIC (Mininet Experiments)

> Build a **reliable transport** on top of UDP, then study **congestion control** (baseline windowing, **TCP Reno**, **TCP CUBIC**) under varying delay/loss and concurrent flows. Includes a Mininet harness to reproduce measurements and check end-to-end integrity.

---

## 1) Problem & Objectives

**Problem.** UDP provides no ordering, reliability, or congestion control. For controlled experiments (and learning), we need a user-space transport that:

1. reliably delivers a file end-to-end over lossy/reordered paths,
2. adapts its sending rate to network conditions, and
3. is testable for **fairness** when multiple flows share a bottleneck.

**Objectives.**

-   Add **segmentation + sequencing + cumulative ACKs** with a **sliding window** on UDP.
-   Detect loss via **timeouts** (RTT-based) and **duplicate ACKs**; trigger **fast retransmit** (+ optional fast recovery).
-   Implement two congestion controllers on the same reliability core:
    -   **Baseline**: fixed window (reference).
    -   **TCP Reno**: slow start, congestion avoidance, fast retransmit/recovery.
    -   **TCP CUBIC**: cubic window growth with loss-epoch memory.
-   Provide a **Mininet** experiment harness that varies **loss** and **delay**, verifies **file integrity** (hash), and reports performance; plus a **dumbbell fairness** test with two simultaneous flows.

---

## 2) Approach — How the Protocol Works (conceptual)

### A) Connection bootstrap & RTT sampling

-   The receiver initiates with a small **handshake** (e.g., `START`), and the sender immediately replies, measuring a **sample RTT**.
-   Sender maintains **EstimatedRTT** and **devRTT** with exponential smoothing (`α, β`) and sets **RTO** as:
    \[
    \text{RTO} = \text{EstimatedRTT} + 4 \cdot \text{devRTT}
    \]
-   **Karn’s rule**: RTT samples from retransmitted packets are discarded to avoid biasing RTO.

### B) Reliable delivery on top of UDP

-   The file is segmented with an **MSS** (e.g., 1400B payload) and tagged with a **monotonic sequence number**.
-   Packets are serialized as **JSON** (sequence, length, payload), with payload encoded to preserve binary content.
-   The receiver implements **cumulative ACKs**: it ACKs the **highest contiguous** sequence seen; out-of-order segments are not advanced (keeps sender logic simple and loss-driven).
-   The sender keeps a **window** of unacknowledged segments; each in-flight segment is tracked (timestamp + bytes) until acknowledged.

### C) Loss detection & retransmission

-   **Timeout path**: if an ACK for the **earliest unacked** segment doesn’t arrive by **RTO**, retransmit that segment and backoff as needed.
-   **Duplicate-ACK path**: on **≥3 duplicate ACKs**, trigger **fast retransmit** of the missing segment (lowest seq outstanding).  
    Optional **fast recovery** keeps the pipe full by temporarily inflating the window around the loss event (Reno semantics).

### D) Congestion control variants

-   **Baseline (fixed window)**: a constant in-flight limit is enforced. Useful as a ground truth for reliability and for isolating the effect of congestion control.
-   **TCP Reno**:
    -   **Slow Start**: start with small `cwnd` and **double per RTT** (ACK-clocked growth) until `ssthresh`.
    -   **Congestion Avoidance**: **+1 MSS per RTT** beyond `ssthresh`.
    -   **On loss**:
        -   Timeout or 3-DupACKs → `ssthresh = cwnd/2`;  
            fast retransmit; enter **fast recovery**; then continue in avoidance.
-   **TCP CUBIC**:
    -   Track **Wmax** (window before last loss) and **loss epoch time**.
    -   Window evolves as:
        \[
        \text{cwnd}(t) = C \cdot (t - K)^3 + W*\text{max},\quad
        K = \sqrt[3]{\frac{W*\text{max}\cdot \beta}{C}}
        \]
        with constants **\(C\)** (scales growth) and **\(\beta\)** (multiplicative decrease factor).
    -   On loss: set **Wmax ← last cwnd**, reduce cwnd by \(\beta\), and start a new epoch; growth then follows the cubic curve (fast ramp-up away from Wmax, gentle near it).

### E) Integrity & observability

-   Receiver writes to an output file and the harness computes **MD5** (or similar) to verify **bit-exact** delivery vs the source.
-   Sender logs **RTO updates**, **loss/dupACK events**, **(fast) retransmissions**, and **throughput/duration** per run.

---

## 3) Experiment Methodology (Mininet)

-   **Topology (single flow)**: two hosts connected by a single bottleneck link; **delay** and **loss** are configured with `tc` (via Mininet’s `TCLink`).
-   **Topology (fairness)**: **dumbbell** with two senders → single bottleneck → two receivers; two flows start in close succession to test **bandwidth sharing**.
-   **Parameter sweeps**: the harness iterates an internal grid of **loss%** and **delay (ms)** for multiple trials, runs a full transfer each time, and aggregates results.
-   **Checks**: after each run, verify **file checksum**; report **time / throughput**.  
    (If you see non-matching hashes, that indicates a protocol bug; the harness prints both hashes to help debug.)

---

## 4) Run It

### Prerequisites

-   **Ubuntu** recommended, **Python ≥ 3.8**, **Mininet** (with `sudo`), `tc`, and `iproute2`.

```bash
sudo apt-get update
sudo apt-get install -y mininet python3 python3-pip
# If Mininet is not available from apt on your distro, install from source: http://mininet.org/download/
```
