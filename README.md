# Artifact for CCS2026 B Submiss 303

## Overview

This artifact accompanies the paper **“DualMS 2.0: Practical Lattice-Based Two-Round Fiat–Shamir
Multi-Signature with Better Efficiency”** (CCS 2026, B Submission 303).<br>

## Artifact Contents

The artifact contains two components:

1. `sage/`: SageMath scripts for the parameter and size results in Tables 1, 6, and 8.
   - `table1.sage` outputs the Table 1 size comparison (run `sage table1.sage`).
   - `table6.sage` outputs the Table 6 summary&nbsp;&nbsp;(run `sage table6.sage`).
     - `dualms.sage`, `dualms-rounding.sage`, `dualms2.0-non-rounding.sage`, and `dualms2.0-rounding.sage`: scheme data.
     - `dualms_common.sage`, `lwe_estimator.py`: shared helpers and MLWE estimates functions.
   - `table8.sage` outputs the structured Table 8 WAN summary (run `sage table8.sage`).
   - `wan-benchmark-n8.sage`: Supporting scripts of Table 8 WAN experiment parameters.
2. `*-benchmark/`: Go benchmarks for DualMS and rounded DualMS2.0.
   - `bin/`: prebuilt Linux binary.
   - `internal/core/`: scheme implementation.
   - `internal/local/`: local benchmark.
   - `network/`: WAN benchmark.

## Requirements

- Linux AMD64 runs the prebuilt Go binaries directly.
- Go 1.19 or later is required to build or run the Go source. Dependencies are
  declared in `go.mod`.
- SageMath is required for the Table 1, Table 6, and Table 8 scripts.

## Benchmark

Experiments ran on Ubuntu 24.04.3 on a laptop with an AMD Ryzen 7 6800H (8 cores, 4.3 GHz) and 32 GB RAM, using single CPU core.

**Results may vary across hardware, environments, and network conditions.**

### 1. Sage

> **Tables 1 and 6:** These scripts present the parameter and size results in Tables 1 and 6.

Run:

```bash
cd sage
sage table1.sage
sage table6.sage
```

---

### 2. Local Benchmark

> **Table 7:** This benchmark presents the local performance results summarized in Table 7.

Run:

```bash
cd dualms-benchmark
./bin/dualms bench 50 32
./bin/dualms bench 50 1024

cd ../dualms2.0-benchmark
./bin/sec3rounding bench 50 32
./bin/sec3rounding bench 50 1024
```

Benchmark command format:

```bash
./bin/dualms bench <runs> <signers>
./bin/sec3rounding bench <runs> <signers>
```

`runs` is the number of benchmark repetitions, and `signers` selects the parameter set. The default rejection and attempt parameters are used.

To compile and run from source:

```bash
go run . bench <runs> <signers>
```

---

### 3. WAN Benchmark
> WAN benchmark performance may vary with hardware specifications, network latency, and the runtime environment.

#### 3.1 The WAN experiment data for Table 8 are:

1. **DualMS:** `dualms-benchmark/network/wan-performance-table8/table8-summary.json`
2. **Rounded DualMS2.0:** `dualms2.0-benchmark/network/wan-performance-table8/table8-summary.json`
```bash
cd sage
sage table8.sage
```

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
