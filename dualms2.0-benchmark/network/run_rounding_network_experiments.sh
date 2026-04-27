#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  ./run_rounding_network_experiments.sh [runs]

Description:
  Orchestrate the Rounding01 n=8 / profile-8 network experiment from the HK server.
  The script launches:
    - HK: coordinator + signer 1
    - US1: signer 2, signer 3
    - NL: signer 4, signer 5
    - UAE: signer 6, signer 7

Defaults:
  runs=10

Environment overrides:
  HK_WORKDIR           default: $HOME/sec3rounding
  REMOTE_WORKDIR       default: $HOME/sec3rounding
  BINARY_NAME          default: bin/sec3rounding
  HK_CONFIG_TEMPLATE   default: network/net8.rounding.vps.hk.json
  REMOTE_CONFIG_TEMPLATE
                       default: network/net8.rounding.vps.remote.json
  HK_RUNTIME_CONFIG    default: net8.rounding.runtime.hk.json
  REMOTE_RUNTIME_CONFIG
                       default: net8.rounding.runtime.remote.json
  SSH_KEY              default: $HOME/.ssh/id_ed25519_sec3
  SSH_USER             default: root
  HK_HOST              default: 47.76.63.156
  US1_HOST             default: 104.194.67.180
  US1_PORT             default: 22
  NL_HOST              default: 45.87.43.76
  NL_PORT              default: 61665
  UAE_HOST             default: 139.185.38.183
  UAE_PORT             default: 22
  RUN_TIMEOUT_SECONDS  default: 600
  OUTPUT_ROOT          default: $HK_WORKDIR/network-runs

Notes:
  - This script assumes the HK server can SSH into the three remote servers
    using the configured SSH key.
  - The config is fixed to the network-only n=8 profile with target_rejection_mn = 5.
  - Each run uses a distinct message, so repeated runs do not replay the same
    online signing instance.
  - The coordinator summaries already include per-run abort counts and
    coordinator traffic (bytes_in / bytes_out).
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

RUNS="${1:-10}"
if ! [[ "$RUNS" =~ ^[0-9]+$ ]] || [[ "$RUNS" -le 0 ]]; then
  echo "runs must be a positive integer" >&2
  exit 1
fi

HK_WORKDIR="${HK_WORKDIR:-$HOME/sec3rounding}"
REMOTE_WORKDIR="${REMOTE_WORKDIR:-$HOME/sec3rounding}"
BINARY_NAME="${BINARY_NAME:-bin/sec3rounding}"
HK_CONFIG_TEMPLATE="${HK_CONFIG_TEMPLATE:-network/net8.rounding.vps.hk.json}"
REMOTE_CONFIG_TEMPLATE="${REMOTE_CONFIG_TEMPLATE:-network/net8.rounding.vps.remote.json}"
HK_RUNTIME_CONFIG="${HK_RUNTIME_CONFIG:-net8.rounding.runtime.hk.json}"
REMOTE_RUNTIME_CONFIG="${REMOTE_RUNTIME_CONFIG:-net8.rounding.runtime.remote.json}"
SSH_KEY="${SSH_KEY:-$HOME/.ssh/id_ed25519_sec3}"
SSH_USER="${SSH_USER:-root}"
HK_HOST="${HK_HOST:-47.76.63.156}"
US1_HOST="${US1_HOST:-104.194.67.180}"
US1_PORT="${US1_PORT:-22}"
NL_HOST="${NL_HOST:-45.87.43.76}"
NL_PORT="${NL_PORT:-61665}"
UAE_HOST="${UAE_HOST:-139.185.38.183}"
UAE_PORT="${UAE_PORT:-22}"
RUN_TIMEOUT_SECONDS="${RUN_TIMEOUT_SECONDS:-600}"
OUTPUT_ROOT="${OUTPUT_ROOT:-$HK_WORKDIR/network-runs}"

SSH_OPTS=(
  -i "$SSH_KEY"
  -o BatchMode=yes
  -o StrictHostKeyChecking=accept-new
  -o ConnectTimeout=10
)

timestamp() {
  date '+%Y%m%d-%H%M%S'
}

require_local_file() {
  local path="$1"
  local desc="$2"
  if [[ ! -e "$path" ]]; then
    echo "missing $desc: $path" >&2
    exit 1
  fi
}

host_port() {
  local host="$1"
  case "$host" in
    "$US1_HOST")
      printf '%s\n' "$US1_PORT"
      ;;
    "$NL_HOST")
      printf '%s\n' "$NL_PORT"
      ;;
    "$UAE_HOST")
      printf '%s\n' "$UAE_PORT"
      ;;
    *)
      printf '%s\n' "22"
      ;;
  esac
}

check_remote_ready() {
  local host="$1"
  local port
  port="$(host_port "$host")"
  ssh -p "$port" "${SSH_OPTS[@]}" "${SSH_USER}@${host}" \
    "test -x '$REMOTE_WORKDIR/$BINARY_NAME'" >/dev/null
}

render_runtime_config() {
  local template="$1"
  local output="$2"
  local message="$3"
  sed -E 's|"message"[[:space:]]*:[[:space:]]*"[^"]*"|"message": "'"$message"'"|' \
    "$template" >"$output"
}

install_remote_runtime_config() {
  local host="$1"
  local local_config="$2"
  local port
  port="$(host_port "$host")"
  scp -P "$port" "${SSH_OPTS[@]}" "$local_config" \
    "${SSH_USER}@${host}:$REMOTE_WORKDIR/$REMOTE_RUNTIME_CONFIG" >/dev/null
}

remote_cleanup() {
  local host="$1"
  local port
  local ids=("${@:2}")
  port="$(host_port "$host")"
  local cmd="pkill -f '$BINARY_NAME net coordinator $HK_RUNTIME_CONFIG' >/dev/null 2>&1 || true;"
  cmd+=" pkill -f '$BINARY_NAME net coordinator $HK_CONFIG_TEMPLATE' >/dev/null 2>&1 || true;"
  for signer_id in "${ids[@]}"; do
    cmd+=" pkill -f '$BINARY_NAME net signer ${signer_id} $REMOTE_RUNTIME_CONFIG' >/dev/null 2>&1 || true;"
    cmd+=" pkill -f '$BINARY_NAME net signer ${signer_id} $REMOTE_CONFIG_TEMPLATE' >/dev/null 2>&1 || true;"
    cmd+=" pkill -f '$BINARY_NAME net signer ${signer_id} $HK_RUNTIME_CONFIG' >/dev/null 2>&1 || true;"
    cmd+=" pkill -f '$BINARY_NAME net signer ${signer_id} $HK_CONFIG_TEMPLATE' >/dev/null 2>&1 || true;"
  done
  ssh -p "$port" "${SSH_OPTS[@]}" "${SSH_USER}@${host}" "$cmd" >/dev/null 2>&1 || true
}

local_cleanup() {
  pkill -f "$BINARY_NAME net coordinator $HK_RUNTIME_CONFIG" >/dev/null 2>&1 || true
  pkill -f "$BINARY_NAME net coordinator $HK_CONFIG_TEMPLATE" >/dev/null 2>&1 || true
  pkill -f "$BINARY_NAME net signer 1 $HK_RUNTIME_CONFIG" >/dev/null 2>&1 || true
  pkill -f "$BINARY_NAME net signer 1 $HK_CONFIG_TEMPLATE" >/dev/null 2>&1 || true
}

start_local_process() {
  local __pid_var="$1"
  local log_file="$2"
  shift 2
  (
    cd "$HK_WORKDIR"
    exec "./$BINARY_NAME" "$@"
  ) >"$log_file" 2>&1 < /dev/null &
  printf -v "$__pid_var" '%s' "$!"
}

start_remote_process() {
  local __pid_var="$1"
  local host="$2"
  local log_file="$3"
  local port
  shift 3
  port="$(host_port "$host")"
  ssh -p "$port" "${SSH_OPTS[@]}" "${SSH_USER}@${host}" \
    "cd '$REMOTE_WORKDIR' && exec './$BINARY_NAME' $*" \
    >"$log_file" 2>&1 < /dev/null &
  printf -v "$__pid_var" '%s' "$!"
}

append_summary_lines() {
  local src_log="$1"
  local dst_file="$2"
  if [[ -f "$src_log" ]]; then
    grep '^NETWORK_SUMMARY ' "$src_log" | sed 's/^NETWORK_SUMMARY //' >>"$dst_file" || true
  fi
}

wait_for_pid_with_timeout() {
  local pid="$1"
  local timeout_seconds="$2"
  if timeout "${timeout_seconds}s" tail --pid="$pid" -f /dev/null >/dev/null 2>&1; then
    wait "$pid"
    return $?
  fi
  kill "$pid" >/dev/null 2>&1 || true
  wait "$pid" >/dev/null 2>&1 || true
  return 124
}

print_run_debug_tail() {
  local run_dir="$1"
  echo "Last coordinator log lines:"
  tail -n 80 "$run_dir/coordinator.log" 2>/dev/null || true
  echo
  echo "Last signer log lines:"
  for log_file in "$run_dir"/*.log; do
    if [[ -f "$log_file" && "$log_file" != "$run_dir/coordinator.log" ]]; then
      echo "--- ${log_file##*/} ---"
      tail -n 30 "$log_file" 2>/dev/null || true
    fi
  done
}

cleanup_all_hosts() {
  local_cleanup
  remote_cleanup "$US1_HOST" 2 3
  remote_cleanup "$NL_HOST" 4 5
  remote_cleanup "$UAE_HOST" 6 7
}

require_local_file "$HK_WORKDIR/$BINARY_NAME" "binary"
require_local_file "$HK_WORKDIR/$HK_CONFIG_TEMPLATE" "HK config template"
require_local_file "$HK_WORKDIR/$REMOTE_CONFIG_TEMPLATE" "remote config template"
require_local_file "$SSH_KEY" "SSH key"

mkdir -p "$OUTPUT_ROOT"

echo "Checking remote hosts..."
check_remote_ready "$US1_HOST"
check_remote_ready "$NL_HOST"
check_remote_ready "$UAE_HOST"
echo "Remote hosts are ready."

RUN_ROOT="$OUTPUT_ROOT/rounding01-n8-$(timestamp)"
mkdir -p "$RUN_ROOT"
COORD_SUMMARIES="$RUN_ROOT/coordinator_summaries.jsonl"
ALL_SUMMARIES="$RUN_ROOT/all_summaries.jsonl"

echo "Logs will be stored in: $RUN_ROOT"

trap cleanup_all_hosts EXIT

for run in $(seq 1 "$RUNS"); do
  run_dir="$RUN_ROOT/run-$(printf '%03d' "$run")"
  mkdir -p "$run_dir"

  echo "=== Run $run/$RUNS ==="
  cleanup_all_hosts
  sleep 1

  run_message="rounding01-vps-network-demo-n8-profile8-$(basename "$RUN_ROOT")-run-$(printf '%03d' "$run")"
  hk_runtime_config_path="$run_dir/$HK_RUNTIME_CONFIG"
  remote_runtime_config_path="$run_dir/$REMOTE_RUNTIME_CONFIG"
  render_runtime_config "$HK_WORKDIR/$HK_CONFIG_TEMPLATE" "$hk_runtime_config_path" "$run_message"
  render_runtime_config "$HK_WORKDIR/$REMOTE_CONFIG_TEMPLATE" "$remote_runtime_config_path" "$run_message"
  cp "$hk_runtime_config_path" "$HK_WORKDIR/$HK_RUNTIME_CONFIG"
  install_remote_runtime_config "$US1_HOST" "$remote_runtime_config_path"
  install_remote_runtime_config "$NL_HOST" "$remote_runtime_config_path"
  install_remote_runtime_config "$UAE_HOST" "$remote_runtime_config_path"
  echo "Run message: $run_message"

  coord_log="$run_dir/coordinator.log"
  hk_signer_log="$run_dir/hk-signer1.log"
  us1_s2_log="$run_dir/us1-signer2.log"
  us1_s3_log="$run_dir/us1-signer3.log"
  nl_s4_log="$run_dir/nl-signer4.log"
  nl_s5_log="$run_dir/nl-signer5.log"
  uae_s6_log="$run_dir/uae-signer6.log"
  uae_s7_log="$run_dir/uae-signer7.log"

  start_local_process coord_pid "$coord_log" net coordinator "$HK_RUNTIME_CONFIG"
  sleep 1
  start_local_process hk_signer_pid "$hk_signer_log" net signer 1 "$HK_RUNTIME_CONFIG"
  start_remote_process us1_s2_pid "$US1_HOST" "$us1_s2_log" net signer 2 "$REMOTE_RUNTIME_CONFIG"
  start_remote_process us1_s3_pid "$US1_HOST" "$us1_s3_log" net signer 3 "$REMOTE_RUNTIME_CONFIG"
  start_remote_process nl_s4_pid "$NL_HOST" "$nl_s4_log" net signer 4 "$REMOTE_RUNTIME_CONFIG"
  start_remote_process nl_s5_pid "$NL_HOST" "$nl_s5_log" net signer 5 "$REMOTE_RUNTIME_CONFIG"
  start_remote_process uae_s6_pid "$UAE_HOST" "$uae_s6_log" net signer 6 "$REMOTE_RUNTIME_CONFIG"
  start_remote_process uae_s7_pid "$UAE_HOST" "$uae_s7_log" net signer 7 "$REMOTE_RUNTIME_CONFIG"

  coord_status=0
  if ! wait_for_pid_with_timeout "$coord_pid" "$RUN_TIMEOUT_SECONDS"; then
    coord_status=$?
  fi

  if [[ "$coord_status" -eq 124 ]]; then
    echo "Run $run timed out after ${RUN_TIMEOUT_SECONDS}s. See $run_dir" >&2
    cleanup_all_hosts
    for pid in \
      "$hk_signer_pid" "$us1_s2_pid" "$us1_s3_pid" "$nl_s4_pid" "$nl_s5_pid" "$uae_s6_pid" "$uae_s7_pid"
    do
      kill "$pid" >/dev/null 2>&1 || true
      wait "$pid" >/dev/null 2>&1 || true
    done
    print_run_debug_tail "$run_dir" >&2
    exit 1
  elif [[ "$coord_status" -ne 0 ]]; then
    echo "Run $run failed. See $coord_log" >&2
    cleanup_all_hosts
    for pid in \
      "$hk_signer_pid" "$us1_s2_pid" "$us1_s3_pid" "$nl_s4_pid" "$nl_s5_pid" "$uae_s6_pid" "$uae_s7_pid"
    do
      wait "$pid" || true
    done
    print_run_debug_tail "$run_dir" >&2
    exit "$coord_status"
  fi

  for pid in \
    "$hk_signer_pid" "$us1_s2_pid" "$us1_s3_pid" "$nl_s4_pid" "$nl_s5_pid" "$uae_s6_pid" "$uae_s7_pid"
  do
    wait "$pid"
  done

  cleanup_all_hosts

  append_summary_lines "$coord_log" "$COORD_SUMMARIES"
  for src_log in \
    "$coord_log" "$hk_signer_log" "$us1_s2_log" "$us1_s3_log" \
    "$nl_s4_log" "$nl_s5_log" "$uae_s6_log" "$uae_s7_log"
  do
    append_summary_lines "$src_log" "$ALL_SUMMARIES"
  done

  tail -n 5 "$coord_log" || true
  echo
  sleep 1
done

echo
echo "Completed $RUNS run(s)."
echo "Coordinator summaries: $COORD_SUMMARIES"
echo "All summaries: $ALL_SUMMARIES"
