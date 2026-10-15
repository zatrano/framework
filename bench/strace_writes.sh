#!/usr/bin/env bash
# Count write and writev syscalls for one plaintext response.
# The response path should be a single write or writev on the connection.
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
go build -o "${TMPDIR:-/tmp}/ztfb" .
trace="${TMPDIR:-/tmp}/ztfb-writes.txt"
taskset -c 0 strace -f -e trace=write,writev -o "$trace" "${TMPDIR:-/tmp}/ztfb" serve zatrano 127.0.0.1:18181 &
pid=$!
cleanup() {
  kill -9 "$pid" 2>/dev/null || true
  # strace may leave the server if it was not the same process group.
  pkill -9 -f 'ztfb serve zatrano 127.0.0.1:18181' 2>/dev/null || true
  wait "$pid" 2>/dev/null || true
}
trap cleanup EXIT
# ss watches the listen socket. A TCP probe is itself a request, and rawhttp
# answers it with a 400 write that is not the response under test.
ready=0
for _ in $(seq 1 100); do
  if ss -ltn | grep -q ':18181'; then
    ready=1
    break
  fi
  sleep 0.1
done
if [[ "$ready" -ne 1 ]]; then
  echo "server did not listen"
  exit 1
fi
printf 'GET /plaintext HTTP/1.1\r\nHost: 127.0.0.1\r\nConnection: close\r\n\r\n' | nc -w 2 127.0.0.1 18181 >/tmp/ztfb-resp.txt || true
sleep 0.2
cleanup
trap - EXIT
echo "---- response head ----"
head -c 200 /tmp/ztfb-resp.txt || true
echo
echo "---- writes ----"
# Drop stdout/stderr (fd 1 and 2). The connection fd is everything else.
grep -E 'writev?\(' "$trace" | grep -Ev 'writev?\((1|2),' | tee /tmp/ztfb-conn-writes.txt || true
n=$(grep -Ec 'writev?\(' /tmp/ztfb-conn-writes.txt || true)
echo "connection writes: ${n:-0}"
if [[ "${n:-0}" -gt 1 ]]; then
  echo "more than one write/writev on the connection"
  exit 1
fi
if [[ "${n:-0}" -lt 1 ]]; then
  echo "no connection write recorded"
  exit 1
fi

# Syscall mix for the bare carriers at the connection levels where rawhttp
# lost the RSS and p99 gate. Summary only; the one-write check above stays.
syscall_summary() {
  local server="$1" conns="$2"
  local summary="${TMPDIR:-/tmp}/ztfb-strace-${server}-${conns}.txt"
  taskset -c 0 strace -c -f -o "$summary" "$bin" serve "$server" 127.0.0.1:18181 &
  local pid=$!
  local ready=0
  for _ in $(seq 1 100); do
    if ss -ltn | grep -q ':18181'; then
      ready=1
      break
    fi
    sleep 0.1
  done
  if [[ "$ready" -ne 1 ]]; then
    echo "syscall: $server did not listen"
    kill -9 "$pid" 2>/dev/null || true
    wait "$pid" 2>/dev/null || true
    return 1
  fi
  taskset -c 1 "$bin" load 127.0.0.1:18181 "GET /plaintext" "$conns" 2 || true
  # strace -c writes the table when the tracee exits. SIGTERM to strace
  # itself can hang while it detaches the Go threads, so stop the server.
  pkill -TERM -P "$pid" 2>/dev/null || true
  for _ in $(seq 1 100); do
    kill -0 "$pid" 2>/dev/null || break
    sleep 0.1
  done
  if kill -0 "$pid" 2>/dev/null; then
    echo "syscall: strace did not exit for $server c=$conns"
    pkill -9 -P "$pid" 2>/dev/null || true
    kill -9 "$pid" 2>/dev/null || true
  fi
  wait "$pid" 2>/dev/null || true
  echo "==== strace -c $server c=$conns ===="
  cat "$summary" || true
}
bin="${TMPDIR:-/tmp}/ztfb"
syscall_summary rawhttp 64
syscall_summary fasthttp 64
syscall_summary rawhttp 256
syscall_summary fasthttp 256
