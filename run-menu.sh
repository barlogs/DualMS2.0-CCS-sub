#!/usr/bin/env bash
# Interactive menu for DualMS 2.0 CCS AE artifact (Tables 1/6/7/8).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT"

# Prefer a conda/Miniforge Sage if present
for _cand in \
  "${SAGE_PREFIX:-$HOME/miniforge}/bin" \
  "$HOME/mambaforge/bin" \
  "$HOME/miniconda3/bin" \
  "$HOME/anaconda3/bin"
do
  if [[ -x "$_cand/sage" ]]; then
    export PATH="$_cand:$PATH"
    break
  fi
done
unset _cand

pause() {
  echo
  read -r -p "Press Enter to return to the menu..." _
}

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "[!] Missing command: $1"
    return 1
  fi
}

run_sage() {
  local script="$1"
  local sage_dir="$ROOT/sage"
  if ! command -v sage >/dev/null 2>&1; then
    echo "[!] SageMath (sage) is not installed or not on PATH."
    echo "    Run: bash ./setup-deps.sh"
    return 1
  fi
  # Sage 10 may leave __file__ set; relative load() then chdirs wrongly.
  (cd "$sage_dir" && sage -c "import os; os.chdir(r'"'"'$sage_dir'"'"'); globals().pop('"'"'__file__'"'"', None); load('"'"'$script'"'"')")
}

ensure_bins() {
  chmod +x "$ROOT/dualms-benchmark/bin/dualms" 2>/dev/null || true
  chmod +x "$ROOT/dualms2.0-benchmark/bin/sec3rounding" 2>/dev/null || true
}

run_local_bench() {
  local scheme="$1" runs="$2" signers="$3"
  ensure_bins
  if [[ "$scheme" == "dualms" ]]; then
    local bin="$ROOT/dualms-benchmark/bin/dualms"
    if [[ -x "$bin" ]]; then
      (cd "$ROOT/dualms-benchmark" && ./bin/dualms bench "$runs" "$signers")
    else
      need_cmd go || return 1
      (cd "$ROOT/dualms-benchmark" && go run . bench "$runs" "$signers")
    fi
  else
    local bin="$ROOT/dualms2.0-benchmark/bin/sec3rounding"
    if [[ -x "$bin" ]]; then
      (cd "$ROOT/dualms2.0-benchmark" && ./bin/sec3rounding bench "$runs" "$signers")
    else
      need_cmd go || return 1
      (cd "$ROOT/dualms2.0-benchmark" && go run . bench "$runs" "$signers")
    fi
  fi
}

print_shipped_wan() {
  echo
  if command -v sage >/dev/null 2>&1; then
    (cd "$ROOT/sage" && sage table8.sage) || true
  else
    if command -v python3 >/dev/null 2>&1; then
      (cd "$ROOT/sage" && python3 table8.sage) || {
        echo "[!] Could not run table8.sage with python3 either."
        return 1
      }
    else
      echo "[!] Need sage or python3 to format Table 8 summaries."
      return 1
    fi
  fi
}

menu_size() {
  while true; do
    clear
    cat <<'MENU'
============================================================
 1) Paper sizes / bit security  (Tables 1 & 6)
============================================================
  [1] Run Table 1  (sizes)          — sage table1.sage
  [2] Run Table 6  (bit security)   — sage table6.sage
  [3] Run both Table 1 and Table 6
  [0] Back to main menu
============================================================
MENU
    read -r -p "Choose: " c
    case "$c" in
      1) echo; run_sage table1.sage; pause ;;
      2) echo; run_sage table6.sage; pause ;;
      3) echo; run_sage table1.sage; echo; run_sage table6.sage; pause ;;
      0) return ;;
      *) echo "Invalid choice."; sleep 1 ;;
    esac
  done
}

menu_local() {
  local runs=50
  while true; do
    clear
    cat <<MENU
============================================================
 2) Local benchmark  (Table 7)
============================================================
 +----------------------------------------------------------+
 | NOTE                                                     |
 | Paper machine: Ubuntu 24.04, AMD Ryzen 7 6800H,          |
 | 32 GB RAM, single core. Absolute timings WILL differ     |
 | on other hardware.                                       |
 +----------------------------------------------------------+

 Default runs = ${runs}  (paper-style). Change with [r].

  [1] DualMS                 n=32
  [2] DualMS                 n=1024
  [3] Rounded DualMS 2.0     n=32
  [4] Rounded DualMS 2.0     n=1024
  [5] Run all four above
  [r] Set repetition count (current: ${runs})
  [0] Back to main menu
============================================================
MENU
    read -r -p "Choose: " c
    case "$c" in
      1) echo; run_local_bench dualms "$runs" 32; pause ;;
      2) echo; run_local_bench dualms "$runs" 1024; pause ;;
      3) echo; run_local_bench dualms2 "$runs" 32; pause ;;
      4) echo; run_local_bench dualms2 "$runs" 1024; pause ;;
      5)
        echo
        run_local_bench dualms "$runs" 32
        run_local_bench dualms "$runs" 1024
        run_local_bench dualms2 "$runs" 32
        run_local_bench dualms2 "$runs" 1024
        pause
        ;;
      r|R)
        read -r -p "New runs (positive int): " runs
        [[ "$runs" =~ ^[1-9][0-9]*$ ]] || runs=50
        ;;
      0) return ;;
      *) echo "Invalid choice."; sleep 1 ;;
    esac
  done
}

menu_wan() {
  while true; do
    clear
    cat <<'MENU'
============================================================
 3) WAN experiment Table 8
============================================================
 We provide the WAN experiment data.

 +----------------------------------------------------------+
 | NOTE                                                     |
 | Source: packaged WAN experiment data                     |
 | (formats the shipped JSON into a table).                 |
 +----------------------------------------------------------+

  [1] Show formatted Table 8
  [0] Back to main menu
============================================================
MENU
    read -r -p "Choose: " c
    case "$c" in
      1) echo; print_shipped_wan; pause ;;
      0) return ;;
      *) echo "Invalid choice."; sleep 1 ;;
    esac
  done
}

main_menu() {
  while true; do
    clear
    cat <<'MENU'
============================================================
 DualMS 2.0 — CCS 2026 Artifact menu
============================================================
 Pick a number and press Enter.

  [1] Paper sizes / bit security     (Tables 1 & 6)
  [2] Local benchmark                (Table 7)
  [3] WAN experiment Table 8
  [0] Exit
============================================================
MENU
    read -r -p "Choose: " c
    case "$c" in
      1) menu_size ;;
      2) menu_local ;;
      3) menu_wan ;;
      0) echo "Bye."; exit 0 ;;
      *) echo "Invalid choice."; sleep 1 ;;
    esac
  done
}

main_menu
