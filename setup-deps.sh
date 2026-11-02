#!/usr/bin/env bash
# Install SageMath (+ common AE deps) for this artifact on Linux.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PREFIX="${SAGE_PREFIX:-$HOME/miniforge}"
MF_URL="https://github.com/conda-forge/miniforge/releases/latest/download/Miniforge3-Linux-x86_64.sh"

echo "==> Target prefix: $PREFIX"

if [[ ! -x "$PREFIX/bin/conda" ]]; then
  echo "==> Installing Miniforge..."
  curl -fsSL "$MF_URL" -o /tmp/miniforge-install.sh
  bash /tmp/miniforge-install.sh -b -p "$PREFIX"
fi

# shellcheck disable=SC1091
source "$PREFIX/etc/profile.d/conda.sh"
conda activate base

echo "==> Installing SageMath from conda-forge (large download)..."
conda install -y -c conda-forge sage

echo "==> Checking Go (local benchmarks)..."
if ! command -v go >/dev/null 2>&1; then
  echo "[!] go not found. Install Go 1.19+ for building from source."
  echo "    Prebuilt Linux AMD64 binaries under bin/ may still work."
else
  go version
fi

echo "==> Linking sage onto PATH hint"
echo "Add to your shell:  source \"$PREFIX/etc/profile.d/conda.sh\" && conda activate base"
echo "Or:  export PATH=\"$PREFIX/bin:\$PATH\""

"$PREFIX/bin/sage" --version || true
echo "==> Done. From artifact root:  bash ./run-menu.sh"
