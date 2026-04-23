# How It Works: In-Depth Technical Specification

This document provides a comprehensive, component-level analysis of the Controlled Execution Sandbox architecture, isolation layers, and security triggers.

---

## 1. System Architecture Overview
The system follows a **Controller-Agent** model. The Controller (Go API) acts as the trusted orchestrator, while the Agent (Hardened Docker Container) is the untrusted execution environment.

### Core Components:
- **API Controller:** Built with Go's standard library. Manages the lifecycle of every execution request.
- **Docker Engine:** The underlying virtualization layer providing kernel-level isolation via Namespaces and Cgroups.
- **Active Traps:** A dual-layer deception system (Filesystem and Network) that triggers immediate termination upon unauthorized access.

---

## 2. The Execution Lifecycle (Step-by-Step)
When a user submits code via the dashboard, the following sequence occurs:

1.  **Request Reception:** The API receives a JSON payload containing `language` and `code`.
2.  **Honeypot Preparation:**
    -   A unique **Linux FIFO (Named Pipe)** is created in the host's `/tmp` directory.
    -   A goroutine is spawned to perform a blocking `Open` on this pipe. This routine waits for a "Reader" event.
3.  **Container Provisioning:**
    -   A `ContainerCreate` call is sent to the Docker Socket.
    -   The `HostConfig` is populated with strict security and resource metadata (see Section 3).
    -   The host's FIFO pipe is bind-mounted as a read-only file at `/var/run/secrets/database.txt`.
4.  **Execution & Monitoring:**
    -   The container starts. Three concurrent "watchers" (using Go channels) monitor for the first finish signal:
        -   **Exit Signal:** Script finished execution normally.
        -   **Timer Signal:** 5-second execution deadline exceeded.
        -   **Trap Signal:** The FIFO pipe watcher unblocked (indicating a read attempt).
5.  **Teardown & Audit:**
    -   Regardless of the outcome, the container is forcefully killed and removed.
    -   The FIFO pipe is deleted from the host.
    -   Logs are extracted, parsed for security triggers, and appended to `execution.log`.

---

## 3. Security Isolation Layers

### A. Resource Constraints (Cgroups)
We enforce strict limits to prevent Denial of Service (DoS) attacks:
- **Memory Limit (64MB):** Prevents memory exhaustion attacks.
- **CPU Limit (0.5 Shares):** Prevents CPU-bound scripts from starving the host.
- **PIDs Limit (20):** Prevents "Fork Bombs" (where a script tries to crash the kernel by spawning infinite processes).

### B. Privilege Minimization
- **User Namespace:** The script runs as `USER 1000:1000`, a non-privileged account.
- **Capability Dropping:** We use `CapDrop: ["ALL"]`. This removes all 40+ Linux root capabilities (like `CAP_NET_ADMIN` or `CAP_SYS_ADMIN`), making the root user inside the container almost powerless.
- **No-New-Privileges:** The `no-new-privileges` flag is set to `true`, preventing the execution of `setuid` binaries that could lead to privilege escalation.

### C. Filesystem Protection
- **Read-Only Root:** The entire OS filesystem inside the container is mounted as `ReadOnly`. Even if an attacker finds a way to write to a system directory, the kernel will block it.
- **Ephemeral Storage:** No persistent volumes are mounted except for the read-only trap file.

---

## 4. Deception & Detection Logic

### FIFO Pipe Trap (Filesystem)
Standard files are passive; you have to poll them to know if they were accessed. We use a **FIFO (First-In, First-Out) pipe** because it is a blocking primitive.
- Inside Go, the `os.OpenFile(path, os.O_WRONLY, 0)` call blocks execution until another process opens the other end of the pipe for reading.
- This allows for **instantaneous detection** with zero CPU overhead for polling. The moment a script runs `cat` or `open()`, the Go watcher wakes up and kills the container.

### API Trap (Network)
The system runs a dummy HTTP server on the host gateway.
- While the container has no internet access, it can see the host gateway.
- Any attempt to perform a network scan or "phone home" to a metadata service results in an immediate trigger.

---

## 5. Audit Logging Mechanism
Every event is recorded in a standardized format in `execution.log`:
`[TIMESTAMP] Lang: [LANG], Status: [STATUS], Duration: [TIME], ExitCode: [CODE]`

- **Success:** Script finished within limits without touching traps.
- **TimedOut:** Script exceeded the 5s window.
- **Terminated(file):** Script attempted to read the honeypot file.
- **Terminated(api):** Script attempted unauthorized network communication.
