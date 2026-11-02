# Artifact for CCS2026 B Submission 303

## Overview

This artifact accompanies the paper **“DualMS 2.0: Practical Lattice-Based Two-Round Fiat–Shamir
Multi-Signature with Better Efficiency”** (CCS 2026, B Submission 303).<br>

## Artifact Contents

```text
. (artifact root)
├── run-menu.sh                 # One-click menu for all experiments (Tables 1, 6, 7, 8)
├── run-menu.cmd                # Windows launcher (via Git Bash / WSL)
├── setup-deps.sh               # Installs SageMath (Miniforge) and checks Go
├── Artifact Appendix.pdf       # CCS artifact appendix
├── README.md
├── sage/                       # SageMath scripts (Tables 1, 6, 8)
│   ├── table1.sage             # Table 1 size comparison
│   ├── table6.sage             # Table 6 parameters / bit security
│   ├── table8.sage             # Formats the packaged WAN summaries into Table 8
│   ├── dualms*.sage            # Scheme data used by table1/table6
│   ├── dualms_common.sage      # Shared helpers
│   ├── lwe_estimator.py        # MLWE estimator
│   └── wan-benchmark-n8.sage   # Table 8 WAN experiment parameters
├── dualms-benchmark/           # Go benchmark: DualMS
│   ├── bin/dualms              # Prebuilt Linux AMD64 binary
│   ├── main.go, go.mod
│   ├── internal/core/          # Scheme implementation
│   ├── internal/local/         # Local benchmark (Table 7)
│   └── network/                # WAN benchmark code, configs, runner script
│       └── wan-performance-table8/
│           ├── table8-summary.json   # DualMS row of Table 8 (10 runs)
│           └── run-001 … run-010/    # Per-run coordinator/signer logs
└── dualms2.0-benchmark/        # Go benchmark: rounded DualMS2.0
    ├── bin/sec3rounding        # Prebuilt Linux AMD64 binary
    ├── main.go, go.mod
    ├── internal/core/
    ├── internal/local/
    └── network/
        └── wan-performance-table8/
            ├── table8-summary.json   # DualMS2.0 row of Table 8 (20 runs)
            └── run-001 … run-020/
```

## Requirements

- Linux AMD64 runs the prebuilt Go binaries directly.
- Go 1.19 or later is required to build or run the Go source. Dependencies are
  declared in `go.mod`.
- SageMath is required for the Table 1, Table 6, and Table 8 scripts.
- To install SageMath, run `./setup-deps.sh`.

## Benchmark

Experiments ran on Ubuntu 24.04.3 on a laptop with an AMD Ryzen 7 6800H (8 cores, 4.3 GHz) and 32 GB RAM, using single CPU core.

**Results may vary across hardware, environments, and network conditions.**

### Quick start

From the artifact root, run:

```bash
./run-menu.sh
```

```text
[1] Paper sizes / bit security     (Tables 1 & 6)
[2] Local benchmark                (Table 7)
[3] WAN experiment Table 8
[0] Exit
```

---

### 1. Size / bit security levels (Tables 1 and 6)

Run `./run-menu.sh`, choose `[1]`, then pick Table 1, Table 6, or both.

Outputs should match Tables 1 and 6.

---

### 2. Local benchmark (Table 7)

Run `./run-menu.sh`, choose `[2]`, then select DualMS / DualMS2.0 with n=32 or 1024, or run all four (50 runs by default; `r` changes it).

Compare latencies to Table 7. Absolute numbers vary by CPU; relative DualMS vs. DualMS2.0 trends should align.

Manual command: `./bin/dualms bench <runs> <signers>` in `dualms-benchmark/`, or `./bin/sec3rounding bench <runs> <signers>` in `dualms2.0-benchmark/`.

---

### 3. WAN benchmark (Table 8)

#### 3.1 Packaged WAN data

Run `./run-menu.sh`, choose `[3]`, then `[1]` to format the packaged WAN summaries into Table 8:

1. **DualMS:** `dualms-benchmark/network/wan-performance-table8/table8-summary.json` (10 runs)
2. **Rounded DualMS2.0:** `dualms2.0-benchmark/network/wan-performance-table8/table8-summary.json` (20 runs)

Per-run logs are in the `run-xxx/` folders next to each summary. Traffic is reported in MiB.

> **WAN experiment note:** The Netherlands VPS used in the paper's Table 8 WAN setup (signers 4 and 5) has expired and is no longer available. Therefore, the WAN experiment results of the DualMS scheme come from a later, separate experiment: the Netherlands signer server was dropped, and signers were redistributed to HK (coordinator + signer 1), US1 (signers 2–4), and UAE (signers 5–7). Latency and traffic thus differ from Table 8.

<div style="height: 3px; background-color: #3b82f6; margin: 24px 0;"></div>

#### 3.2 WAN Experiment deployment

The WAN scripts run the n=8 active-signer experiment across one coordinator server and three remote signer servers.

Each benchmark's `network/net8.*.json` files are configuration templates and should be adjusted to the target hosts.

Each `network/table8_summary.py` formats the coordinator JSONL output; the WAN runner invokes it after completing the runs.

Deploy each benchmark directory on the coordinator server, and place the corresponding binary under `bin/` on all servers. The coordinator script uses `HK_WORKDIR` and `REMOTE_WORKDIR` to locate the coordinator and remote deployments respectively.

---

Before running, adjust the two JSON configuration files and the WAN script for each scheme:

- DualMS: `dualms-benchmark/network/net8.dualms.vps.hk.json`, `dualms-benchmark/network/net8.dualms.vps.remote.json`, and `dualms-benchmark/network/run_dualms_network_experiments.sh`.
- Rounded DualMS2.0: `dualms2.0-benchmark/network/net8.rounding.vps.hk.json`, `dualms2.0-benchmark/network/net8.rounding.vps.remote.json`, and `dualms2.0-benchmark/network/run_rounding_network_experiments.sh`.

The HK and remote JSON templates differ only in `coordinator_public`: the coordinator uses `127.0.0.1:7100`, while remote signers use the HK public address. Set the remote hosts, SSH user and key, ports, and deployment directories in the script.

Run on the coordinator server:

```bash
cd ccs2026-artifacts/dualms-benchmark
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/dualms .
HK_WORKDIR="$PWD" REMOTE_WORKDIR="$PWD" ./network/run_dualms_network_experiments.sh 10

cd ccs2026-artifacts/dualms2.0-benchmark
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/sec3rounding .
HK_WORKDIR="$PWD" REMOTE_WORKDIR="$PWD" ./network/run_rounding_network_experiments.sh 10
```

Each execution creates a timestamped directory under `network-runs/` on the coordinator server and writes the run logs and JSONL summaries there.

## Disclaimer

This proof-of-concept artifact is only used for the artifact evaluation of CCS 2026 B Submission 303 and is not intended for any production deployment.
