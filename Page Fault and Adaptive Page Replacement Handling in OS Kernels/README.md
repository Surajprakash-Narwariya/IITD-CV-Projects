# xv6 — Memory Printer & Adaptive Swapping (Assignment 3)

> Adds a **Ctrl+I Memory Printer** and a **disk-backed paging system** with an **adaptive eviction policy** on xv6.  
> Focus: stability under memory pressure, fast visibility into per-process memory, and tunable trade-offs.

---

## 1) Problem & Goals

**Problem.** Vanilla xv6 offers no paging: once free frames are exhausted, the system can panic or stall. It also lacks a quick, low-overhead way to see how much memory each user process is consuming.

**Goals.**

-   Provide an **instant, non-intrusive memory snapshot** of active processes (usable during any workload).
-   Add **swapping** so workloads can keep running when RAM is tight.
-   Use an **adaptive policy** to balance reclaim burstiness, page-fault rate, and user-perceived latency.

---

## 2) Approach — How It Works (conceptual, not code)

### A) Memory Printer (Ctrl+I)

-   **Trigger path.** A console shortcut (**Ctrl+I**) raises a lightweight kernel action that does **not** kill or reschedule the foreground job.
-   **What it prints.** For each active user process, the kernel reports **PID → resident user pages** (i.e., frames currently mapped in user space).
-   **Why this design.** Walking process metadata is fast and side-effect-free; it gives an immediate sense of pressure, fragmentation, and growth without functionally perturbing the workload.

### B) Swapping Architecture (Disk-backed Pages)

-   **Swap area.** A dedicated **swap region** on disk is sliced into **fixed-size slots**; each slot holds exactly one page.
-   **Page state.** A page is either **resident (present)** in RAM or **swapped** (its PTE encodes the swap-slot ID and “not present”).
-   **Eviction trigger.** When **free pages fall below a threshold**, the system evicts a **batch** of pages to swap to refill the freelist.

### C) Choosing What to Evict (Two-level choice)

1. **Process choice.** Pressure is applied first to processes with **larger resident sets**—they contribute most to memory pressure.
2. **Page choice.** Within a chosen process, prefer pages that are **not recently accessed** (approximate “cold-first”) to reduce immediate refaults.

**Eviction mechanics.**

-   Write the page to a **free swap slot**, update its PTE to **not-present + slot location**, free the physical frame, and flush the mapping.
-   The process continues running; if it later touches the evicted page, a regular **page fault** will bring it back.

### D) Page Fault (Swap-in) Path

1. **Detect.** On a not-present fault, the kernel identifies the faulting virtual address.
2. **Ensure frame.** If the freelist is empty, perform **targeted eviction** of a few more pages before retrying.
3. **Restore.** Read the page from its slot into a free frame, rebuild its present PTE with the original permissions, release the slot.
4. **Resume.** Execution returns to the faulting instruction transparently to user code.

### E) Adaptive Eviction Policy (α / β)

-   **Threshold (Th).** The free-page watermark that triggers reclamation.
-   **Batch size (Npg).** How many pages to evict per trigger.
-   **Adaptation.** After each trigger:
    -   Increase **Npg** slightly (by **α%**) to reduce how often we interrupt workloads if pressure persists.
    -   Decrease **Th** slightly (by **β%**) so the system stabilizes around the **actual** working-set size rather than thrashing above it.
-   **Intuition.** α tunes **burstiness vs. trigger frequency**; β tunes **how aggressively** the system runs near low free-page levels.

### F) Safety, Correctness, Observability

-   **Safety.** Eviction never targets kernel pages; slot accounting guarantees no double allocation; metadata updates are ordered to avoid dangling mappings.
-   **Correctness.** Only fully written swap-out pages are marked non-present; swap-in restores the original access permissions.
-   **Observability.** The kernel logs **threshold**, **batch size**, and **swap-in/out events**; the Memory Printer reflects **current residency** in real time.

---

## 3) Build, Run, and Try It

> **Prerequisites (Ubuntu recommended)**
>
> ```bash
> sudo apt-get update
> sudo apt-get install -y build-essential gdb qemu-system-x86
> ```
>
> **macOS (Apple Silicon):** QEMU via Homebrew or run inside a Linux VM.

### Build & Launch

```bash
# 1) Clone
git clone https://github.com/<your-user>/<your-repo>.git
cd <your-repo>

# 2) (Optional) Configure policy
# - Keep single-core for reproducible behavior
# - Tune α / β for eviction (burstiness vs. responsiveness)
# - Use a smaller PHYSTOP during testing to force pressure and observe swapping

# 3) Build & run
make clean && make
make qemu-nox
```
