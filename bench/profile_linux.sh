#!/usr/bin/env bash
# Short Linux profiles for the cells the matrix left open.
# Server on CPU 0, load generator on the rest. GOMAXPROCS=1 for the server.
# Prints gctrace. Writes alloc and goroutine profiles. Does not set GOGC.
set -euo pipefail
bench_dir="$(cd "$(dirname "$0")" && pwd)"
# Resolve before cd. A relative PROFILE_OUT would otherwise land under ztfb
# and the artifact upload would find an empty directory.
if [[ -n "${PROFILE_OUT:-}" ]]; then
  mkdir -p "$PROFILE_OUT"
  out="$(cd "$PROFILE_OUT" && pwd)"
else
  out="$bench_dir/results/linux/profile"
  mkdir -p "$out"
fi
cd "$bench_dir/ztfb"
export GOTOOLCHAIN="${GOTOOLCHAIN:-go1.26.0}"
export GOWORK=off
export GOFLAGS=
export APP_ENV=production
export APP_DEBUG=false
export LOG_LEVEL=error
export CORS_ENABLED=true
export APP_KEY=zatrano-ci-test-key-not-prod!!!!
export GOMAXPROCS=1
go build -o "${TMPDIR:-/tmp}/ztfb" .
bin="${TMPDIR:-/tmp}/ztfb"
ncpu="$(nproc)"
client_cpus="1-$((ncpu - 1))"
if [[ "$ncpu" -lt 2 ]]; then
  client_cpus="0"
fi

run_case() {
  local server="$1" route="$2" conns="$3" seconds="${4:-8}" codec="${5:-encodingjson}" do_trace="${6:-}"
  local tag
  tag="$(printf '%s-%s-%s-%s' "$codec" "$server" "$route" "$conns" | tr ' /{}' '-----')"
  local trace="$out/${tag}.gctrace"
  local pid
  local trace_path=""
  if [[ "$do_trace" == "trace" ]]; then
    trace_path="$out/${tag}.trace"
  fi
  GODEBUG=gctrace=1 \
  ZTFB_CODEC="$codec" \
  ZTFB_PROFILE_SECONDS="$seconds" \
  ZTFB_CPU_PROFILE="$out/${tag}.cpu.pb" \
  ZTFB_ALLOC_PROFILE="$out/${tag}.alloc.pb" \
  ZTFB_HEAP_PROFILE="$out/${tag}.heap.pb" \
  ZTFB_GOROUTINE_PROFILE="$out/${tag}.goroutine.pb" \
  ZTFB_TRACE="$trace_path" \
  taskset -c 0 "$bin" serve "$server" 127.0.0.1:18181 >"$trace" 2>&1 &
  pid=$!
  local ready=0
  for _ in $(seq 1 100); do
    if ss -ltn | grep -q ':18181'; then
      ready=1
      break
    fi
    sleep 0.1
  done
  if [[ "$ready" -ne 1 ]]; then
    echo "profile: $tag did not listen"
    kill -9 "$pid" 2>/dev/null || true
    wait "$pid" 2>/dev/null || true
    return 1
  fi
  taskset -c "$client_cpus" "$bin" load 127.0.0.1:18181 "$route" "$conns" "$((seconds + 4))" || true
  # The server timer calls the profile flush, then exits. kill -9 is the backstop.
  local i
  for i in $(seq 1 $((seconds + 15))); do
    if ! kill -0 "$pid" 2>/dev/null; then
      break
    fi
    sleep 1
  done
  kill -9 "$pid" 2>/dev/null || true
  wait "$pid" 2>/dev/null || true
  echo "==== $tag gc ===="
  grep -E 'gc |gomaxprocs' "$trace" | tail -n 8 || true
}

# Plaintext 4096 is long enough that a 1-CPU mark phase is inside the trace.
run_case zatrano "GET /plaintext" 4096 40 encodingjson trace
run_case fiber "GET /plaintext" 4096 40 encodingjson trace
run_case zatrano "GET /json" 1024
run_case fiber "GET /json" 1024
run_case zatrano "POST /echo" 64
run_case fiber "POST /echo" 64
run_case zatrano "GET /users/{id}" 64
run_case fiber "GET /users/{id}" 64
for conns in 64 256 1024 4096; do
  run_case zatrano "POST /echo" "$conns" 8 goccy
  run_case fiber "POST /echo" "$conns" 8 goccy
done
for conns in 256 1024 4096; do
  run_case zatrano "GET /users/{id}" "$conns" 8 goccy
  run_case fiber "GET /users/{id}" "$conns" 8 goccy
done
for conns in 64 256 1024 4096; do
  run_case rawhttp "GET /plaintext" "$conns" 8 encodingjson
  run_case fasthttp "GET /plaintext" "$conns" 8 encodingjson
done
echo "profiles in $out"
find "$out" -type f -printf '%s %p\n' | sort
if ! find "$out" -type f | grep -q .; then
  echo "profile directory is empty"
  exit 1
fi
