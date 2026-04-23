# Controlled Execution Sandbox

A high-performance, security-hardened sandbox for executing untrusted Python and Bash code. Built with Go and Docker, featuring active deception mechanisms to detect and neutralize malicious activity.

## Features
- **Hardened Isolation:** Utilizes Linux Namespaces and Cgroups to restrict RAM (64MB), CPU (0.5), and PIDs (20).
- **Minimal Privileges:** Runs as non-root, drops all capabilities, and enforces a read-only root filesystem.
- **Active Traps:** 
    - **FIFO File Trap:** Instant detection of unauthorized reads to sensitive paths.
    - **Internal API Trap:** Traps network scanning and lateral movement attempts.
- **Minimalist Dashboard:** A sleek, professional web interface for real-time monitoring and execution.
- **Audit Logging:** Every execution is logged with detailed status and resource usage.

## Setup
1. **Requirements:**
    - Go 1.22 or higher.
    - Docker Engine running locally.
2. **Installation:**
    ```bash
    go mod tidy
    ```
3. **Running the Server:**
    ```bash
    go run cmd/sandbox-api/main.go
    ```
4. **Accessing the Dashboard:**
    Open `http://localhost:8080` in your browser.

## Project Structure
- `cmd/sandbox-api/`: Entry point and API handlers.
- `internal/executor/`: Docker orchestration and resource limiting.
- `internal/traps/`: Filesystem (FIFO) and Network (API) honeypot logic.
- `static/`: Minimalist frontend assets.
- `execution.log`: Persistent audit trail.

## Security Architecture
Detailed technical specifications of the isolation layers and deception triggers can be found in [how_it_works.md](./how_it_works.md).
