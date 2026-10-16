#!/usr/bin/env bash
# Repeat the Linux paired matrix on a machine you control.
# No arguments. --quick is 5 rounds and 1 run and does not edit the README.
# --allow-any-governor continues when the CPU governor is not performance
# and does not edit the README.
set -euo pipefail

bench_dir="$(cd "$(dirname "$0")" && pwd)"
repo_dir="$(cd "$bench_dir/.." && pwd)"
quick=0
allow_any=0
for arg in "$@"; do
  case "$arg" in
    --quick) quick=1 ;;
    --allow-any-governor) allow_any=1 ;;
    *)
      echo "unknown argument: $arg" >&2
      exit 2
      ;;
  esac
done

if [[ ! -r /etc/os-release ]]; then
  echo "Ubuntu 22.04 or newer is required" >&2
  exit 1
fi
# shellcheck disable=SC1091
. /etc/os-release
if [[ "${ID:-}" != "ubuntu" ]]; then
  echo "Ubuntu 22.04 or newer is required (found ${ID:-unknown})" >&2
  exit 1
fi
major="${VERSION_ID%%.*}"
if [[ "$major" -lt 22 ]]; then
  echo "Ubuntu 22.04 or newer is required (found ${VERSION_ID})" >&2
  exit 1
fi

cores="$(nproc)"
if [[ "$cores" -lt 4 ]]; then
  echo "at least 4 logical cores are required (found $cores)" >&2
  exit 1
fi

gov=""
if [[ -r /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor ]]; then
  gov="$(cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor)"
elif command -v cpupower >/dev/null 2>&1; then
  gov="$(cpupower frequency-info -p 2>/dev/null | awk -F'"' '/governor/ {print $2; exit}')"
fi
if [[ -z "$gov" || "$gov" != "performance" ]]; then
  echo "warning: CPU governor is '${gov:-unreadable}', not performance" >&2
  echo "set it with:" >&2
  echo "  echo performance | sudo tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor" >&2
  echo "or:" >&2
  echo "  sudo cpupower frequency-set -g performance" >&2
  if [[ "$allow_any" -ne 1 ]]; then
    echo "stop. Pass --allow-any-governor to continue without editing the README." >&2
    exit 1
  fi
fi

arch="$(uname -m)"
case "$arch" in
  x86_64) goarch=amd64 ;;
  aarch64) goarch=arm64 ;;
  *)
    echo "unsupported architecture: $arch" >&2
    exit 1
    ;;
esac
if ! command -v go >/dev/null 2>&1 || [[ "$(go env GOVERSION 2>/dev/null || true)" != "go1.26.6" ]]; then
  tarball="go1.26.6.linux-${goarch}.tar.gz"
  url="https://go.dev/dl/${tarball}"
  tmp="$(mktemp)"
  curl -fsSL -o "$tmp" "$url"
  sudo rm -rf /usr/local/go
  sudo tar -C /usr/local -xzf "$tmp"
  rm -f "$tmp"
  export PATH="/usr/local/go/bin:${PATH}"
fi
if [[ "$(go env GOVERSION)" != "go1.26.6" ]]; then
  echo "Go 1.26.6 is required (found $(go env GOVERSION))" >&2
  exit 1
fi
if ! command -v wrk >/dev/null 2>&1; then
  sudo apt-get update
  sudo apt-get install -y wrk
fi

rounds=20
runs=3
if [[ "$quick" -eq 1 ]]; then
  rounds=5
  runs=1
fi

day="$(date +%Y-%m-%d)"
out_dir="$bench_dir/results/dedicated/$day"
mkdir -p "$out_dir"
cpu_model="$(awk -F: '/model name/ {gsub(/^ +/, "", $2); print $2; exit}' /proc/cpuinfo)"
kernel="$(uname -r)"
sha="$(git -C "$repo_dir" rev-parse HEAD)"
{
  echo "cpu_model=$cpu_model"
  echo "cores=$cores"
  echo "kernel=$kernel"
  echo "governor=${gov:-unreadable}"
  echo "go_version=$(go env GOVERSION)"
  echo "wrk_version=$(wrk --version 2>/dev/null | awk 'NR==1 {print; exit}')"
  echo "commit=$sha"
  echo "date=$day"
  echo "rounds=$rounds"
  echo "runs=$runs"
  echo "quick=$quick"
  echo "allow_any_governor=$allow_any"
} > "$out_dir/env.txt"

# The harness pins the server with taskset to CPU 0 and GOMAXPROCS=1,
# and the load generator to the other CPUs. Both frameworks use that split.
export GOTOOLCHAIN=go1.26.6
export GOWORK=off
export GOFLAGS=

for run in $(seq 1 "$runs"); do
  for spec in "go default" "wrk default" "go pooled" "wrk pooled"; do
    set -- $spec
    client="$1"
    config="$2"
    name="${client}-${config}-${run}"
    echo "run $name rounds=$rounds"
    CLIENT="$client" CONFIG="$config" CODEC=encodingjson PAIR=fiber \
      ROUNDS="$rounds" ROUTES= CONNS= MEASURE= GC_MIN= \
      OUT="results/dedicated/$day/${name}.jsonl" \
      bash "$bench_dir/run_linux.sh"
  done
done

(
  cd "$bench_dir"
  GOTOOLCHAIN=go1.26.6 GOWORK=off go run ./cmd/summarize "$out_dir"
)
echo "wrote $out_dir"
