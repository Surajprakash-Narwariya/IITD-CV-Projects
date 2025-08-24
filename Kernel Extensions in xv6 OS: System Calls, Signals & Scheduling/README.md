# xv6 OS Enhancements — Auth Shell, History, Syscall Blocking, `chmod`, Signals, Priority Scheduler

> **TL;DR**: Combines two xv6 assignments into one repo—adds login, history, syscall controls, `chmod`, keyboard-driven signals, staged starts via `custom_fork`, a profiler, and a tunable priority-boosting scheduler.

---

## Problem → Solution Overview

**Problem.** Stock xv6 lacks (a) login/auth, (b) process history, (c) selective syscall control, (d) UNIX-like permissions, (e) keyboard→signal semantics, and (f) a fair, configurable scheduler with metrics—making it harder to teach/experiment with OS security + scheduling.

**What I built.**

**Assignment-1 (Shell & FS)**

-   Login-gated shell (3 attempts; creds from `Makefile`).
-   `history` command via syscall **ID=22**: lists exec’d processes (pid, name, total memory).
-   `block`/`unblock` syscalls **IDs=23/24**: deny selected syscall IDs for the shell’s descendants (`fork`/`exit` protected).
-   `chmod` syscall **ID=25** with 3-bit mode (r=1, w=2, x=4) enforced by FS and `exec/open/write`.

**Assignment-2 (Signals & Scheduling)**

-   Keyboard signals: **Ctrl+C**→SIGINT kill, **Ctrl+B**→SIGBG suspend, **Ctrl+F**→SIGFG resume, **Ctrl+G**→SIGCUSTOM (user handler).
-   `signal(sighandler_t)` to register a user-space handler for SIGCUSTOM.
-   `custom_fork(start_later, exec_time)` + `scheduler_start()` to stage processes and release together.
-   Scheduler profiler: per-PID **TAT / WT / RT / #CS** on exit.
-   Dynamic priority scheduler (single-core):  
    \[
    \pi_i(t)=\pi_i(0) - \alpha \cdot C_i(t) + \beta \cdot W_i(t)
    \]
    Tunables in `Makefile`: `PI0`, `ALPHA`, `BETA`.

---

## Technical Design (Concise)

-   **Login before shell**: prompt loop in `init.c` reads `USERNAME/PASSWORD` from `Makefile`.
-   **History**: on successful `exec`, record `(pid,name,mem,ctime)`; `sys_gethistory()` copies to user; `history` prints ascending by time.
-   **Syscall blocklist**: per-process bitmap inherited on `fork`; `syscall()` dispatcher rejects blocked IDs with `-1` (never block `fork`/`exit`).
-   **`chmod`**: `inode.mode` defaults to `7` (set in `mkfs`); `sys_chmod(path,mode)` validates 0..7; checks in `open/write/exec`.
-   **Keyboard→signals**: `consoleintr()` detects Ctrl+[CBFG], sets flags; timer/trap delivers signals to eligible procs (skip pid 1/2).
-   **User handlers**: `signal()` stores handler; SIGCUSTOM re-routes to handler in user space.
-   **Staged start**: `custom_fork(start_later, exec_time)` defers run; `scheduler_start()` marks runnable; scheduler decrements `exec_time`.
-   **Profiler**: kernel tracks run/wait timestamps and context switches; prints summary on `exit()`.
-   **Scheduler**: dynamic priority with tie-break (lowest PID); `CPUS=1` enforced.

---

## Build & Run

> **Prereqs (Ubuntu)**  
> `sudo apt-get update && sudo apt-get install -y build-essential gdb qemu-system-x86 expect`

```bash
# 1) Clone
git clone
cd <repo>

# 2) Configure (edit Makefile)
# --- Assignment-1 ---
USERNAME=anand
PASSWORD=pa55word
SYS_gethistory=22
SYS_block=23
SYS_unblock=24
SYS_chmod=25

# --- Assignment-2 ---
CPUS=1        # single core
PI0=100       # πi(0)
ALPHA=1       # CPU penalty
BETA=2        # wait boost

# 3) Build
make clean && make

# 4) Run (no X)
make qemu-nox
```
