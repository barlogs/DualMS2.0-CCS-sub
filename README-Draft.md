# Open Science Artifacts

This directory contains the Sage parameter scripts and Go benchmark artifacts for the proof-of-concept implementations.

## How to Use This Artifact

Download or clone the anonymous artifact from this url:

```text
https://anonymous.4open.science/r/DualMS20Artifact43ED
```

Then enter the artifact directory:

```bash
cd <artifact-directory>
```

This artifact requires SageMath, Python 3, and Go. The following sections describe how to run the Sage parameter scripts, local benchmarks, and WAN benchmarks.

## Sage Parameter Scripts

The following Sage scripts correspond to the parameter settings in Table 6 of the paper:

| Scheme | Sage script | Notebook |
| --- | --- | --- |
| `DualMS` baseline | `sage/dualms.sage` | `sage/dualms.ipynb` |
| `DualMS2.0` without rounding | `sage/dualms2.0-non-rounding.sage` | `sage/dualms2.0-non-rounding.ipynb` |
| `DualMS2.0` with rounding | `sage/dualms2.0-rounding.sage` | `sage/dualms2.0-rounding.ipynb` |

The scripts use `sage/dualms_common.sage` for shared helper functions and `sage/lwe_estimator.py` for MLWE estimates.

Before running, install Python 3 and SageMath.

Run:

```bash
cd openscience/sage
sage dualms.sage
sage dualms2.0-non-rounding.sage
sage dualms2.0-rounding.sage
```

## Benchmark

The benchmark artifacts are pure Go implementations for evaluating the proposed constructions. They require Go 1.19 or later. Lattigo and the other dependencies are declared in `go.mod` and are downloaded by the Go toolchain on first use.

**WARNING:** These implementations are academic proof-of-concept prototypes and are not intended for production deployment.

### Codebase Overview

- `dualms-benchmark/`
  - Go benchmark artifact for the `DualMS` baseline.
  - `main.go`: Command-line entry point for local and WAN benchmarks.
  - `internal/core/`: Core scheme implementation, including parameters, setup, key generation, key aggregation, signing, aggregation, verification, hashing, sampling, norm checks, and ring arithmetic.
  - `internal/local/`: Single-server local benchmark runner.
  - `network/`: WAN benchmark runner, network message codecs, network configuration loader, n=8 configuration templates, and the WAN experiment script.
  - `bin/`: Build-output location for the benchmark binary.

- `dualms2.0-benchmark/`
  - Go benchmark artifact for rounded `DualMS2.0`.
  - `main.go`: Command-line entry point for local and WAN benchmarks.
  - `internal/core/`: Core scheme implementation, including parameters, setup, key generation, key aggregation, signing, rounded aggregation, verification, hashing, sampling, norm checks, and ring arithmetic.
  - `internal/local/`: Single-server local benchmark runner.
  - `network/`: WAN benchmark runner, network message codecs, network configuration loader, n=8 configuration templates, and the WAN experiment script.
  - `bin/`: Build-output location for the benchmark binary.

### Local Benchmark

Run:

```bash
cd openscience/dualms-benchmark
go run . bench 1 32
go run . bench 1 1024

cd openscience/dualms2.0-benchmark
go run . bench 1 32
go run . bench 1 1024
```

Benchmark command format:

```bash
go run . bench <runs> <signers>
```

`runs` is the number of benchmark repetitions, and `signers` selects the parameter set. The default rejection and attempt parameters are used.

### Network Benchmark

The WAN scripts run the n=8 active-signer experiment across one coordinator server and three remote signer servers. The `network/net8.*.json` files are configuration templates and should be adjusted to the target hosts.

Deploy each benchmark directory on the coordinator server, and place the corresponding binary under `bin/` on all servers. The coordinator script uses `HK_WORKDIR` and `REMOTE_WORKDIR` to locate the coordinator and remote deployments.

Run on the coordinator server:

```bash
cd openscience/dualms-benchmark
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/dualms .
HK_WORKDIR="$PWD" REMOTE_WORKDIR="$PWD" ./network/run_dualms_network_experiments.sh 10

cd openscience/dualms2.0-benchmark
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/sec3rounding .
HK_WORKDIR="$PWD" REMOTE_WORKDIR="$PWD" ./network/run_rounding_network_experiments.sh 10
```

Output summaries are written under `network-runs/`.
