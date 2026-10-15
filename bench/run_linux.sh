#!/usr/bin/env bash
# Paired ZATRANO vs Fiber on Linux. The fiber pair also runs fiber-tuned.
# The binding ratio is zatrano/fiber-tuned. zatrano/fiber is the official
# requestid middleware column. rawhttp still compares rawhttp with fasthttp.
# go-plaintext-4096 passes MEASURE=60s and ZTFB_GCTRACE=1. That window is
# fixed for every server. GC cycles inside it are recorded and may be 0.
# GC_MIN is empty on that job, so the run is not extended until 3 cycles.
#
# The server is pinned to CPU 0 and the load generator to the remaining CPUs.
# ztfb does that with sched_setaffinity, the same call taskset uses:
#   taskset -c 0            server (GOMAXPROCS=1)
#   taskset -c 1-$((n-1))   load generator
#
# Absolute requests/sec on a shared runner are noise. Report the paired ratio
# and its 95% bootstrap interval from the jsonl "paired" rows.
#
# Before a run on a machine you control, set the CPU governor:
#   sudo cpupower frequency-set -g performance
# or:
#   echo performance | sudo tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor
#
# Load generator: CLIENT=go (default) or CLIENT=wrk.
# Race check (needs gcc; this is the Linux path, not a Windows skip):
#   (cd "$(dirname "$0")/.." && go test -race ./... -count=1)
# The pull request runs that command in .github/workflows/tests.yml.

set -euo pipefail
bench_dir="$(cd "$(dirname "$0")" && pwd)"
out="${OUT:-$bench_dir/results/linux.jsonl}"
case "$out" in
  /*) ;;
  *) out="$bench_dir/$out" ;;
esac
mkdir -p "$(dirname "$out")"
cd "$bench_dir/ztfb"

if [[ -r /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor ]]; then
  echo "cpu governor: $(cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor)"
else
  echo "cpu governor: unavailable. Report paired ratios, not absolute rps."
fi
ncpu="$(nproc)"
echo "affinity: taskset -c 0 server (GOMAXPROCS=1), taskset -c 1-$((ncpu - 1)) load generator"
echo "report paired ratios only; absolute rps on a shared runner is noise"

export GOTOOLCHAIN="${GOTOOLCHAIN:-go1.26.0}"
export GOWORK=off
export GOFLAGS=
go test -count=1 .
go build -o "${TMPDIR:-/tmp}/ztfb" .

args=(run -o "$out" -rounds "${ROUNDS:-20}" -client "${CLIENT:-go}" -config "${CONFIG:-default}" -codec "${CODEC:-encodingjson}" -pair "${PAIR:-fiber}")
if [[ -n "${ROUTES:-}" ]]; then
  args+=(-routes "$ROUTES")
fi
if [[ -n "${CONNS:-}" ]]; then
  args+=(-conns "$CONNS")
fi
if [[ -n "${MEASURE:-}" ]]; then
  args+=(-measure "$MEASURE")
fi
if [[ -n "${GC_MIN:-}" ]]; then
  args+=(-gc-min "$GC_MIN")
fi
exec "${TMPDIR:-/tmp}/ztfb" "${args[@]}"
