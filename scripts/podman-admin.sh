#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HOST_CPU_COUNT="$(getconf _NPROCESSORS_ONLN 2>/dev/null || nproc)"
HOST_MEMORY_MIB="$(awk '/MemTotal/ {printf "%d", $2 / 1024}' /proc/meminfo)"
MAX_CPU_CORES="$(awk -v cpus="$HOST_CPU_COUNT" 'BEGIN { max = cpus / 2; if (max < 0.5) max = 0.5; printf "%.3f", max }')"
MAX_CPUSET_CPUS="$((HOST_CPU_COUNT / 2))"
if [[ "$MAX_CPUSET_CPUS" -lt 1 ]]; then
  MAX_CPUSET_CPUS=1
fi
MAX_PROFILE_CPU_PERCENT="$((HOST_CPU_COUNT * 50))"
MAX_MEMORY_MIB="$((HOST_MEMORY_MIB / 2))"

DEFAULT_CPUS="$(awk -v max="$MAX_CPU_CORES" 'BEGIN { if (max >= 2) print "2"; else if (max >= 1) print "1"; else print "0.5" }')"
DEFAULT_CPUSET_COUNT="$(awk -v cpus="$DEFAULT_CPUS" -v max="$MAX_CPUSET_CPUS" 'BEGIN { count = int(cpus); if (cpus > count) count++; if (count < 1) count = 1; if (count > max) count = max; print count }')"
if [[ "$DEFAULT_CPUSET_COUNT" -le 1 ]]; then
  DEFAULT_CPUSET="0"
else
  DEFAULT_CPUSET="0-$((DEFAULT_CPUSET_COUNT - 1))"
fi
DEFAULT_BUILD_JOBS="$DEFAULT_CPUSET_COUNT"

CPUSET="${PODMAN_ADMIN_CPUSET:-$DEFAULT_CPUSET}"
CPUS="${PODMAN_ADMIN_CPUS:-$DEFAULT_CPUS}"
CPU_PERIOD="${PODMAN_ADMIN_CPU_PERIOD:-100000}"
CPU_QUOTA="${PODMAN_ADMIN_CPU_QUOTA:-$(awk -v cpus="$CPUS" -v period="$CPU_PERIOD" 'BEGIN { printf "%.0f", cpus * period }')}"
BUILD_JOBS="${PODMAN_ADMIN_BUILD_JOBS:-$DEFAULT_BUILD_JOBS}"
MAX_CPU_QUOTA="$(awk -v cpus="$MAX_CPU_CORES" -v period="$CPU_PERIOD" 'BEGIN { printf "%.0f", cpus * period }')"

PROFILE_ENABLED="1"
PROFILE_SLICE="${PODMAN_ADMIN_PROFILE_SLICE:-router-ai-atius-podman-admin.slice}"
PROFILE_CPU_QUOTA="${PODMAN_ADMIN_PROFILE_CPU_QUOTA:-${MAX_PROFILE_CPU_PERCENT}%}"
PROFILE_CPU_WEIGHT="${PODMAN_ADMIN_PROFILE_CPU_WEIGHT:-100}"
PROFILE_MEMORY_HIGH="${PODMAN_ADMIN_PROFILE_MEMORY_HIGH:-$((HOST_MEMORY_MIB * 45 / 100))M}"
PROFILE_MEMORY_MAX="${PODMAN_ADMIN_PROFILE_MEMORY_MAX:-${MAX_MEMORY_MIB}M}"
PROFILE_MEMORY_SWAP_MAX="${PODMAN_ADMIN_PROFILE_MEMORY_SWAP_MAX:-0}"
PROFILE_TASKS_MAX="${PODMAN_ADMIN_PROFILE_TASKS_MAX:-8192}"

MEMORY_RESERVATION="${PODMAN_ADMIN_MEMORY_RESERVATION:-$PROFILE_MEMORY_HIGH}"
MEMORY_MAX="${PODMAN_ADMIN_MEMORY_MAX:-$PROFILE_MEMORY_MAX}"
MEMORY_SWAP="${PODMAN_ADMIN_MEMORY_SWAP:-$MEMORY_MAX}"

COMPOSE_FILE="${PODMAN_ADMIN_COMPOSE_FILE:-$ROOT_DIR/podman-compose.yml}"
COMPOSE_PROVIDER="${PODMAN_ADMIN_COMPOSE_PROVIDER:-$(command -v podman-compose 2>/dev/null || true)}"
POD_NAME="${PODMAN_ADMIN_POD_NAME:-atius-ai-router}"
CONTAINER_NAME="${PODMAN_ADMIN_CONTAINER_NAME:-router-ai-atius}"
EXPECTED_CONTAINERS_RAW="${PODMAN_ADMIN_EXPECTED_CONTAINERS:-router-ai-atius redis postgres}"
UNIT_NAME="${PODMAN_ADMIN_UNIT_NAME:-container-router-ai-atius.service}"
CLIANYTHING="${PODMAN_ADMIN_CLIANYTHING:-$ROOT_DIR/bin/clianything}"
OMNI_CLI_DIR="${PODMAN_ADMIN_OMNI_CLI_DIR:-/home/ubuntu/GitHub/omni-srv-admin/cli}"
read -r -a EXPECTED_CONTAINERS <<<"$EXPECTED_CONTAINERS_RAW"

usage() {
  cat <<'EOF'
Usage: scripts/podman-admin.sh <command> [args...]

Commands:
  limits
  status
  inspect-limits
  verify-runtime-limits
  verify-profile
  verify-container-cgroups
  profile-run [command...]
  run-container [podman run args...]
  build [podman build args...]
  compose-raw [podman-compose args...]
  compose-up [--build] [services...]
  compose-down [services...]
  compose-build [services...]
  build-image [image] [dockerfile] [context]
  cli [clianything args...]
  omni-run <profile> -- <command...>
  prod-restart
  verify
EOF
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 3
  fi
}

fail_policy() {
  echo "resource policy violation: $*" >&2
  exit 2
}

numeric_lte() {
  awk -v value="$1" -v max="$2" 'BEGIN { exit !(value > 0 && value <= max) }'
}

memory_to_bytes() {
  local value="$1"
  case "$value" in
    *G|*g)
      awk -v value="${value%[Gg]}" 'BEGIN { printf "%.0f", value * 1024 * 1024 * 1024 }'
      ;;
    *M|*m)
      awk -v value="${value%[Mm]}" 'BEGIN { printf "%.0f", value * 1024 * 1024 }'
      ;;
    *K|*k)
      awk -v value="${value%[Kk]}" 'BEGIN { printf "%.0f", value * 1024 }'
      ;;
    *)
      printf '%s\n' "$value"
      ;;
  esac
}

cpuset_count() {
  awk -v cpuset="$1" 'BEGIN {
    n = split(cpuset, parts, ",")
    for (i = 1; i <= n; i++) {
      if (parts[i] ~ /^[0-9]+-[0-9]+$/) {
        split(parts[i], range, "-")
        count += range[2] - range[1] + 1
      } else if (parts[i] ~ /^[0-9]+$/) {
        count += 1
      } else {
        count = -1
        break
      }
    }
    print count
  }'
}

validate_policy() {
  numeric_lte "$CPUS" "$MAX_CPU_CORES" ||
    fail_policy "PODMAN_ADMIN_CPUS=${CPUS} exceeds 50% host CPU (${MAX_CPU_CORES})"
  numeric_lte "$CPU_QUOTA" "$MAX_CPU_QUOTA" ||
    fail_policy "PODMAN_ADMIN_CPU_QUOTA=${CPU_QUOTA} exceeds 50% host CPU quota (${MAX_CPU_QUOTA})"
  numeric_lte "$BUILD_JOBS" "$MAX_CPUSET_CPUS" ||
    fail_policy "PODMAN_ADMIN_BUILD_JOBS=${BUILD_JOBS} exceeds 50% host CPU workers (${MAX_CPUSET_CPUS})"

  local cpu_count
  cpu_count="$(cpuset_count "$CPUSET")"
  [[ "$cpu_count" -gt 0 && "$cpu_count" -le "$MAX_CPUSET_CPUS" ]] ||
    fail_policy "PODMAN_ADMIN_CPUSET=${CPUSET} selects ${cpu_count} CPUs; max is ${MAX_CPUSET_CPUS}"

  [[ "$PROFILE_CPU_QUOTA" == *% ]] ||
    fail_policy "PODMAN_ADMIN_PROFILE_CPU_QUOTA must be a percentage"
  local profile_percent="${PROFILE_CPU_QUOTA%\%}"
  numeric_lte "$profile_percent" "$MAX_PROFILE_CPU_PERCENT" ||
    fail_policy "PODMAN_ADMIN_PROFILE_CPU_QUOTA=${PROFILE_CPU_QUOTA} exceeds ${MAX_PROFILE_CPU_PERCENT}%"

  local max_memory_bytes profile_high_bytes profile_max_bytes profile_swap_bytes
  local memory_reservation_bytes memory_max_bytes memory_swap_bytes
  max_memory_bytes="$((MAX_MEMORY_MIB * 1024 * 1024))"
  profile_high_bytes="$(memory_to_bytes "$PROFILE_MEMORY_HIGH")"
  profile_max_bytes="$(memory_to_bytes "$PROFILE_MEMORY_MAX")"
  profile_swap_bytes="$(memory_to_bytes "$PROFILE_MEMORY_SWAP_MAX")"
  memory_reservation_bytes="$(memory_to_bytes "$MEMORY_RESERVATION")"
  memory_max_bytes="$(memory_to_bytes "$MEMORY_MAX")"
  memory_swap_bytes="$(memory_to_bytes "$MEMORY_SWAP")"

  numeric_lte "$profile_high_bytes" "$max_memory_bytes" ||
    fail_policy "PODMAN_ADMIN_PROFILE_MEMORY_HIGH=${PROFILE_MEMORY_HIGH} exceeds 50% host memory"
  numeric_lte "$profile_max_bytes" "$max_memory_bytes" ||
    fail_policy "PODMAN_ADMIN_PROFILE_MEMORY_MAX=${PROFILE_MEMORY_MAX} exceeds 50% host memory"
  [[ "$profile_swap_bytes" == "0" ]] ||
    fail_policy "PODMAN_ADMIN_PROFILE_MEMORY_SWAP_MAX must be 0 for the hard 50% profile"
  numeric_lte "$memory_reservation_bytes" "$max_memory_bytes" ||
    fail_policy "PODMAN_ADMIN_MEMORY_RESERVATION=${MEMORY_RESERVATION} exceeds 50% host memory"
  numeric_lte "$memory_max_bytes" "$max_memory_bytes" ||
    fail_policy "PODMAN_ADMIN_MEMORY_MAX=${MEMORY_MAX} exceeds 50% host memory"
  numeric_lte "$memory_swap_bytes" "$max_memory_bytes" ||
    fail_policy "PODMAN_ADMIN_MEMORY_SWAP=${MEMORY_SWAP} exceeds 50% host memory"
  [[ "$profile_high_bytes" -le "$profile_max_bytes" ]] ||
    fail_policy "profile memory high must be <= profile memory max"
  [[ "$memory_reservation_bytes" -le "$memory_max_bytes" ]] ||
    fail_policy "container memory reservation must be <= memory max"
  [[ "$memory_swap_bytes" -le "$memory_max_bytes" ]] ||
    fail_policy "container memory swap total must be <= memory max"
}

reject_limit_args() {
  local context="$1"
  shift
  local arg
  for arg in "$@"; do
    case "$arg" in
      --cpus|--cpus=*|--cpuset-cpus|--cpuset-cpus=*|--cpu-period|--cpu-period=*|--cpu-quota|--cpu-quota=*|\
      --memory|-m|--memory=*|-m=*|--memory-reservation|--memory-reservation=*|--memory-swap|--memory-swap=*|\
      --podman-run-args|--podman-run-args=*|--podman-build-args|--podman-build-args=*)
        fail_policy "${context} cannot override CPU or memory limits (${arg})"
        ;;
    esac
  done
}

reject_direct_podman_workload() {
  local context="$1"
  shift
  if [[ "${1:-}" == "podman" ]]; then
    case "${2:-}" in
      build|run|compose)
        fail_policy "${context} must use podman-admin build/run-container/compose commands for Podman workloads"
        ;;
    esac
  fi
}

validate_compose_provider() {
  if [[ -z "$COMPOSE_PROVIDER" ]]; then
    fail_policy "podman-compose provider not found; set PODMAN_ADMIN_COMPOSE_PROVIDER"
  fi
  if [[ "$COMPOSE_PROVIDER" != */* ]]; then
    COMPOSE_PROVIDER="$(command -v "$COMPOSE_PROVIDER" 2>/dev/null || true)"
  fi
  [[ -x "$COMPOSE_PROVIDER" ]] ||
    fail_policy "compose provider is not executable: ${COMPOSE_PROVIDER}"
  local help
  help="$("$COMPOSE_PROVIDER" --help 2>&1 || true)"
  [[ "$help" == *"--podman-build-args"* && "$help" == *"--podman-run-args"* ]] ||
    fail_policy "compose provider does not support required podman-compose limit flags: ${COMPOSE_PROVIDER}"
}

current_cgroup_path() {
  awk -F: '$2 == "" { print $3; exit }' /proc/self/cgroup
}

in_admin_profile() {
  local cgroup
  cgroup="$(current_cgroup_path)"
  [[ "$cgroup" == *"/${PROFILE_SLICE}/"* || "$cgroup" == *"/${PROFILE_SLICE}" ]]
}

profile_cpu_max_expected() {
  local quota_percent="${PROFILE_CPU_QUOTA%\%}"
  awk -v quota_percent="$quota_percent" -v period=100000 \
    'BEGIN { printf "%.0f %d", quota_percent * period / 100, period }'
}

print_limits() {
  cat <<EOF
cpu_limit=${CPUS}
cpuset=${CPUSET}
cpu_period=${CPU_PERIOD}
cpu_quota=${CPU_QUOTA}
build_jobs=${BUILD_JOBS}
memory_reservation=${MEMORY_RESERVATION}
memory_max=${MEMORY_MAX}
memory_swap=${MEMORY_SWAP}
host_cpu_count=${HOST_CPU_COUNT}
max_cpu_cores=${MAX_CPU_CORES}
max_cpuset_cpus=${MAX_CPUSET_CPUS}
host_memory_mib=${HOST_MEMORY_MIB}
max_memory_mib=${MAX_MEMORY_MIB}
profile_enabled=${PROFILE_ENABLED}
profile_slice=${PROFILE_SLICE}
profile_cpu_quota=${PROFILE_CPU_QUOTA}
profile_memory_high=${PROFILE_MEMORY_HIGH}
profile_memory_max=${PROFILE_MEMORY_MAX}
profile_memory_swap_max=${PROFILE_MEMORY_SWAP_MAX}
profile_tasks_max=${PROFILE_TASKS_MAX}
compose_provider=${COMPOSE_PROVIDER}
compose_file=${COMPOSE_FILE}
pod_name=${POD_NAME}
container_name=${CONTAINER_NAME}
expected_containers=${EXPECTED_CONTAINERS_RAW}
unit_name=${UNIT_NAME}
EOF
}

run_profiled() {
  if in_admin_profile; then
    "$@"
    return
  fi

  require_cmd systemd-run
  systemd-run \
    --user \
    --scope \
    --quiet \
    --collect \
    --same-dir \
    "--slice=${PROFILE_SLICE}" \
    -p "CPUQuota=${PROFILE_CPU_QUOTA}" \
    -p "CPUWeight=${PROFILE_CPU_WEIGHT}" \
    -p "MemoryHigh=${PROFILE_MEMORY_HIGH}" \
    -p "MemoryMax=${PROFILE_MEMORY_MAX}" \
    -p "MemorySwapMax=${PROFILE_MEMORY_SWAP_MAX}" \
    -p "TasksMax=${PROFILE_TASKS_MAX}" \
    "$@"
}

run_compose() {
  require_cmd podman
  validate_compose_provider
  reject_limit_args "podman compose" "$@"
  local -a cmd=(
    podman compose
    --podman-build-args "--cpuset-cpus=${CPUSET} --cpu-period=${CPU_PERIOD} --cpu-quota=${CPU_QUOTA} --jobs=${BUILD_JOBS} --memory=${MEMORY_MAX} --memory-swap=${MEMORY_SWAP}"
    --podman-run-args "--cpus=${CPUS} --cpuset-cpus=${CPUSET} --memory=${MEMORY_MAX} --memory-reservation=${MEMORY_RESERVATION} --memory-swap=${MEMORY_SWAP}"
    "$@"
  )
  run_profiled env PODMAN_COMPOSE_PROVIDER="$COMPOSE_PROVIDER" "${cmd[@]}"
}

pod_containers() {
  require_cmd podman
  podman ps -a --filter "pod=${POD_NAME}" --format '{{.Names}}'
}

cmd_status() {
  if [[ -x "$CLIANYTHING" ]]; then
    run_profiled "$CLIANYTHING" status
    echo
    run_profiled "$CLIANYTHING" providers --all
    echo
  fi

  require_cmd podman
  podman ps -a --filter "pod=${POD_NAME}" --format 'table {{.Names}}\t{{.Image}}\t{{.Status}}'
  echo
  cmd_inspect_limits
}

cmd_inspect_limits() {
  require_cmd podman
  local found=0
  while IFS= read -r container; do
    [[ -n "$container" ]] || continue
    found=1
    podman inspect --type container "$container" --format \
      'name={{.Name}} running={{.State.Running}} cpuset={{.HostConfig.CpusetCpus}} cpu_quota={{.HostConfig.CpuQuota}} cpu_period={{.HostConfig.CpuPeriod}} nano_cpus={{.HostConfig.NanoCpus}} memory={{.HostConfig.Memory}} memory_reservation={{.HostConfig.MemoryReservation}} memory_swap={{.HostConfig.MemorySwap}}'
  done < <(pod_containers)

  if [[ "$found" -eq 0 ]]; then
    echo "no containers found in pod: ${POD_NAME}" >&2
    exit 2
  fi
}

cmd_verify_runtime_limits() {
  require_cmd podman
  local expected_nano expected_memory expected_memory_reservation expected_memory_swap
  expected_nano="$(awk -v cpus="$CPUS" 'BEGIN { printf "%.0f", cpus * 1000000000 }')"
  expected_memory="$(memory_to_bytes "$MEMORY_MAX")"
  expected_memory_reservation="$(memory_to_bytes "$MEMORY_RESERVATION")"
  expected_memory_swap="$(memory_to_bytes "$MEMORY_SWAP")"
  local failed=0
  local checked=0
  local line name running cpuset cpu_quota cpu_period nano_cpus memory memory_reservation memory_swap
  local -a containers=()
  local -A seen=()
  local -A running_by_name=()

  while IFS= read -r line; do
    [[ -n "$line" ]] || continue
    containers+=("$line")
  done < <(pod_containers)

  if [[ "${#containers[@]}" -eq 0 ]]; then
    echo "no containers found in pod: ${POD_NAME}" >&2
    exit 2
  fi

  while IFS='|' read -r name running cpuset cpu_quota cpu_period nano_cpus memory memory_reservation memory_swap; do
    [[ -n "$name" ]] || continue
    case "$name" in
      *-infra|*-infra-*) continue ;;
    esac
    checked=$((checked + 1))
    seen["$name"]=1
    running_by_name["$name"]="$running"
    if [[ "$running" != "true" ]]; then
      echo "runtime cap failed: ${name} is not running" >&2
      failed=1
    fi
    if [[ "$cpuset" != "$CPUSET" ]]; then
      echo "runtime cap failed: ${name} cpuset=${cpuset:-<empty>} expected=${CPUSET}" >&2
      failed=1
    fi
    if [[ "$nano_cpus" != "$expected_nano" && ! ( "$cpu_quota" == "$CPU_QUOTA" && "$cpu_period" == "$CPU_PERIOD" ) ]]; then
      echo "runtime cap failed: ${name} nano_cpus=${nano_cpus} quota=${cpu_quota}/${cpu_period} expected_nano=${expected_nano} or quota=${CPU_QUOTA}/${CPU_PERIOD}" >&2
      failed=1
    fi
    if [[ "$memory" != "$expected_memory" ]]; then
      echo "runtime cap failed: ${name} memory=${memory:-<empty>} expected=${expected_memory}" >&2
      failed=1
    fi
    if [[ "$memory_reservation" != "$expected_memory_reservation" ]]; then
      echo "runtime cap failed: ${name} memory_reservation=${memory_reservation:-<empty>} expected=${expected_memory_reservation}" >&2
      failed=1
    fi
    if [[ "$memory_swap" != "$expected_memory_swap" ]]; then
      echo "runtime cap failed: ${name} memory_swap=${memory_swap:-<empty>} expected=${expected_memory_swap}" >&2
      failed=1
    fi
  done < <(
    podman inspect --type container "${containers[@]}" --format \
      '{{.Name}}|{{.State.Running}}|{{.HostConfig.CpusetCpus}}|{{.HostConfig.CpuQuota}}|{{.HostConfig.CpuPeriod}}|{{.HostConfig.NanoCpus}}|{{.HostConfig.Memory}}|{{.HostConfig.MemoryReservation}}|{{.HostConfig.MemorySwap}}'
  )

  local expected
  for expected in "${EXPECTED_CONTAINERS[@]}"; do
    if [[ -z "${seen[$expected]:-}" ]]; then
      echo "runtime cap failed: expected container missing from pod ${POD_NAME}: ${expected}" >&2
      failed=1
    elif [[ "${running_by_name[$expected]}" != "true" ]]; then
      echo "runtime cap failed: expected container is not running: ${expected}" >&2
      failed=1
    fi
  done

  if [[ "$checked" -eq 0 ]]; then
    echo "no non-infra containers found in pod: ${POD_NAME}" >&2
    exit 2
  fi
  if [[ "$failed" -ne 0 ]]; then
    exit 2
  fi
  echo "runtime limits OK: ${checked} non-infra containers capped at cpus=${CPUS} memory=${MEMORY_MAX} cpuset=${CPUSET}"
}

cmd_verify_profile() {
  local expected_cpu_max expected_memory_high expected_memory_max expected_memory_swap sample
  expected_cpu_max="$(profile_cpu_max_expected)"
  expected_memory_high="$(memory_to_bytes "$PROFILE_MEMORY_HIGH")"
  expected_memory_max="$(memory_to_bytes "$PROFILE_MEMORY_MAX")"
  expected_memory_swap="$(memory_to_bytes "$PROFILE_MEMORY_SWAP_MAX")"

  # shellcheck disable=SC2016
  sample="$(
    run_profiled bash -c 'set -euo pipefail
cg="$(awk -F: '\''$2=="" { print $3 }'\'' /proc/self/cgroup)"
base="/sys/fs/cgroup${cg}"
printf "cgroup=%s\n" "$cg"
printf "cpu.max=%s\n" "$(cat "$base/cpu.max")"
printf "memory.high=%s\n" "$(cat "$base/memory.high")"
printf "memory.max=%s\n" "$(cat "$base/memory.max")"
printf "memory.swap.max=%s\n" "$(cat "$base/memory.swap.max")"'
  )"
  printf '%s\n' "$sample"

  local observed_cpu_max observed_memory_high observed_memory_max observed_memory_swap
  observed_cpu_max="$(awk -F= '$1 == "cpu.max" { print $2 }' <<<"$sample")"
  observed_memory_high="$(awk -F= '$1 == "memory.high" { print $2 }' <<<"$sample")"
  observed_memory_max="$(awk -F= '$1 == "memory.max" { print $2 }' <<<"$sample")"
  observed_memory_swap="$(awk -F= '$1 == "memory.swap.max" { print $2 }' <<<"$sample")"

  if [[ "$observed_cpu_max" != "$expected_cpu_max" ]]; then
    echo "profile cap failed: cpu.max=${observed_cpu_max:-<empty>} expected=${expected_cpu_max}" >&2
    exit 2
  fi
  if [[ "$observed_memory_high" != "$expected_memory_high" ]]; then
    echo "profile cap failed: memory.high=${observed_memory_high:-<empty>} expected=${expected_memory_high}" >&2
    exit 2
  fi
  if [[ "$observed_memory_max" != "$expected_memory_max" ]]; then
    echo "profile cap failed: memory.max=${observed_memory_max:-<empty>} expected=${expected_memory_max}" >&2
    exit 2
  fi
  if [[ "$observed_memory_swap" != "$expected_memory_swap" ]]; then
    echo "profile cap failed: memory.swap.max=${observed_memory_swap:-<empty>} expected=${expected_memory_swap}" >&2
    exit 2
  fi

  echo "profile limits OK: slice=${PROFILE_SLICE} cpu=${PROFILE_CPU_QUOTA} memory_max=${PROFILE_MEMORY_MAX}"
}

cmd_verify_container_cgroups() {
  require_cmd podman
  local expected_cpu_max expected_memory expected_memory_swap failed=0
  expected_cpu_max="${CPU_QUOTA} ${CPU_PERIOD}"
  expected_memory="$(memory_to_bytes "$MEMORY_MAX")"
  expected_memory_swap="$(memory_to_bytes "$MEMORY_SWAP")"

  local container sample observed_cpu_max observed_cpuset observed_memory observed_swap
  for container in "${EXPECTED_CONTAINERS[@]}"; do
    if ! podman container exists "$container"; then
      echo "container cgroup failed: missing container ${container}" >&2
      failed=1
      continue
    fi
    if [[ "$(podman inspect --type container "$container" --format '{{.State.Running}}')" != "true" ]]; then
      echo "container cgroup failed: container not running ${container}" >&2
      failed=1
      continue
    fi
    sample="$(
      podman exec "$container" sh -c 'set -eu
base=/sys/fs/cgroup
printf "name=%s\n" "$(cat /etc/hostname 2>/dev/null || printf unknown)"
printf "cpu.max=%s\n" "$(cat "$base/cpu.max")"
printf "cpuset.cpus.effective=%s\n" "$(cat "$base/cpuset.cpus.effective")"
printf "memory.max=%s\n" "$(cat "$base/memory.max")"
printf "memory.swap.max=%s\n" "$(cat "$base/memory.swap.max")"'
    )"
    printf 'container=%s\n%s\n' "$container" "$sample"
    observed_cpu_max="$(awk -F= '$1 == "cpu.max" { print $2 }' <<<"$sample")"
    observed_cpuset="$(awk -F= '$1 == "cpuset.cpus.effective" { print $2 }' <<<"$sample")"
    observed_memory="$(awk -F= '$1 == "memory.max" { print $2 }' <<<"$sample")"
    observed_swap="$(awk -F= '$1 == "memory.swap.max" { print $2 }' <<<"$sample")"
    if [[ "$observed_cpu_max" != "$expected_cpu_max" ]]; then
      echo "container cgroup failed: ${container} cpu.max=${observed_cpu_max:-<empty>} expected=${expected_cpu_max}" >&2
      failed=1
    fi
    if [[ "$observed_cpuset" != "$CPUSET" ]]; then
      echo "container cgroup failed: ${container} cpuset=${observed_cpuset:-<empty>} expected=${CPUSET}" >&2
      failed=1
    fi
    if [[ "$observed_memory" != "$expected_memory" ]]; then
      echo "container cgroup failed: ${container} memory.max=${observed_memory:-<empty>} expected=${expected_memory}" >&2
      failed=1
    fi
    if [[ "$observed_swap" == "max" || "$observed_swap" -gt "$expected_memory_swap" ]]; then
      echo "container cgroup failed: ${container} memory.swap.max=${observed_swap:-<empty>} expected<=${expected_memory_swap}" >&2
      failed=1
    fi
  done

  if [[ "$failed" -ne 0 ]]; then
    exit 2
  fi
  echo "container cgroups OK: ${EXPECTED_CONTAINERS_RAW}"
}

cmd_profile_run() {
  if [[ "${1:-}" == "--" ]]; then
    shift
  fi
  if [[ "$#" -eq 0 ]]; then
    echo "usage: scripts/podman-admin.sh profile-run [command...]" >&2
    exit 1
  fi
  reject_direct_podman_workload "profile-run" "$@"
  run_profiled "$@"
}

cmd_compose_up() {
  local build_flag=""
  local -a services=()
  while (($#)); do
    case "$1" in
      --build)
        build_flag="--build"
        ;;
      *)
        services+=("$1")
        ;;
    esac
    shift
  done
  run_compose -f "$COMPOSE_FILE" up -d ${build_flag:+$build_flag} "${services[@]}"
}

cmd_compose_down() {
  run_compose -f "$COMPOSE_FILE" down "$@"
}

cmd_compose_build() {
  run_compose -f "$COMPOSE_FILE" build "$@"
}

cmd_run_container() {
  require_cmd podman
  reject_limit_args "podman run" "$@"
  run_profiled podman run \
    --cpus="${CPUS}" \
    --cpuset-cpus="${CPUSET}" \
    --memory="${MEMORY_MAX}" \
    --memory-reservation="${MEMORY_RESERVATION}" \
    --memory-swap="${MEMORY_SWAP}" \
    "$@"
}

cmd_build() {
  require_cmd podman
  reject_limit_args "podman build" "$@"
  run_profiled podman build \
    --cpuset-cpus="${CPUSET}" \
    --cpu-period="${CPU_PERIOD}" \
    --cpu-quota="${CPU_QUOTA}" \
    --jobs="${BUILD_JOBS}" \
    --memory="${MEMORY_MAX}" \
    --memory-swap="${MEMORY_SWAP}" \
    "$@"
}

cmd_build_image() {
  local image="${1:-localhost/router-ai-atius:cpu-capped-dev}"
  local dockerfile="${2:-Dockerfile.dev}"
  local context="${3:-$ROOT_DIR}"
  cmd_build \
    -f "${dockerfile}" \
    -t "${image}" \
    "${context}"
}

cmd_cli() {
  if [[ ! -x "$CLIANYTHING" ]]; then
    echo "clianything not executable: ${CLIANYTHING}" >&2
    exit 3
  fi
  run_profiled "$CLIANYTHING" "$@"
}

cmd_omni_run() {
  local profile="${1:-}"
  if [[ -z "$profile" ]]; then
    echo "usage: scripts/podman-admin.sh omni-run <profile> -- <command...>" >&2
    exit 1
  fi
  shift
  if [[ "${1:-}" == "--" ]]; then
    shift
  fi
  if [[ "$#" -eq 0 ]]; then
    echo "missing command for omni-run" >&2
    exit 1
  fi
  reject_direct_podman_workload "omni-run" "$@"
  if [[ ! -d "$OMNI_CLI_DIR" ]]; then
    echo "omni CLI dir not found: ${OMNI_CLI_DIR}" >&2
    exit 3
  fi
  run_profiled env PYTHONPATH="$OMNI_CLI_DIR" \
    python3 -m omni srv1-ops resources run "$profile" -- "$@"
}

cmd_prod_restart() {
  require_cmd systemctl
  systemctl --user daemon-reload
  systemctl --user restart "${UNIT_NAME}"
  systemctl --user status "${UNIT_NAME}" --no-pager --lines=0
  echo
  cmd_inspect_limits
  if [[ -x "$CLIANYTHING" ]]; then
    echo
    run_profiled "$CLIANYTHING" status
  fi
}

cmd_verify() {
  "${ROOT_DIR}/scripts/podman-validate.sh" "${COMPOSE_FILE}"
  echo
  run_compose -f "$COMPOSE_FILE" config >/tmp/router-ai-atius-podman-compose.rendered.yml
  echo "rendered_compose=/tmp/router-ai-atius-podman-compose.rendered.yml"
  echo
  cmd_inspect_limits
  echo
  cmd_verify_runtime_limits
  echo
  cmd_verify_profile
  echo
  cmd_verify_container_cgroups
}

main() {
  local command="${1:-}"
  if [[ -z "$command" ]]; then
    usage
    exit 1
  fi
  shift || true

  validate_policy

  case "$command" in
    limits)
      print_limits
      ;;
    status)
      cmd_status
      ;;
    inspect-limits)
      cmd_inspect_limits
      ;;
    verify-runtime-limits)
      cmd_verify_runtime_limits
      ;;
    verify-profile)
      cmd_verify_profile
      ;;
    verify-container-cgroups)
      cmd_verify_container_cgroups
      ;;
    profile-run)
      cmd_profile_run "$@"
      ;;
    run-container)
      cmd_run_container "$@"
      ;;
    build)
      cmd_build "$@"
      ;;
    compose-raw)
      run_compose "$@"
      ;;
    compose-up)
      cmd_compose_up "$@"
      ;;
    compose-down)
      cmd_compose_down "$@"
      ;;
    compose-build)
      cmd_compose_build "$@"
      ;;
    build-image)
      cmd_build_image "$@"
      ;;
    cli)
      cmd_cli "$@"
      ;;
    omni-run)
      cmd_omni_run "$@"
      ;;
    prod-restart)
      cmd_prod_restart "$@"
      ;;
    verify)
      cmd_verify
      ;;
    *)
      usage
      exit 1
      ;;
  esac
}

main "$@"
