#!/usr/bin/env bash
# Short Linux profiles for the cells the matrix left open.
# Server on CPU 0, load generator on the rest. GOMAXPROCS=1 for the server.
# Prints gctrace. Writes alloc and goroutine profiles. Does not set GOGC.
set -euo pipefail
bench_dir="$(cd "$(dirname "$0")" && pwd)"
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
out="${PROFILE_OUT:-$bench_dir/results/linux/profile}"
mkdir -p "$out"
go build -o "${TMPDIR:-/tmp}/ztfb" .
bin="${TMPDIR:-/tmp}/ztfb"
ncpu="$(nproc)"
client_cpus="1-$((ncpu - 1))"
if [[ "$ncpu" -lt 2 ]]; then
  client_cpus="0"
fi

run_case() {
  local server="$1" route="$2" conns="$3"
  local tag
  tag="$(printf '%s-%s-%s' "$server" "$route" "$conns" | tr ' /{}' '----')"
  local trace="$out/${tag}.gctrace"
  local pid
  GODEBUG=gctrace=1 \
  ZTFB_PROFILE_SECONDS=8 \
  ZTFB_ALLOC_PROFILE="$out/${tag}.alloc.pb" \
  ZTFB_GOROUTINE_PROFILE="$out/${tag}.goroutine.pb" \
  taskset -c 0 "$bin" serve "$server" 127.0.0.1:18181 >"$trace" 2>&1 &
  pid=$!
  local ready=0
  for _ in $(seq 1 100); do
    if bash -c 'echo >/dev/tcp/127.0.0.1/18181' 2>/dev/null; then
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
  taskset -c "$client_cpus" "$bin" load 127.0.0.1:18181 "$route" "$conns" 12 || true
  # Let the profile timer exit. Do not leave the listener up.
  local i
  for i in $(seq 1 30); do
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

run_case zatrano "GET /plaintext" 4096
run_case fiber "GET /plaintext" 4096
run_case zatrano "GET /json" 1024
run_case fiber "GET /json" 1024
run_case zatrano "POST /echo" 64
run_case fiber "POST /echo" 64
run_case zatrano "GET /users/{id}" 64
run_case fiber "GET /users/{id}" 64
echo "profiles in $out"
