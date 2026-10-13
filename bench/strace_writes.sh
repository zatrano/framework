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
cleanup() { kill "$pid" 2>/dev/null || true; wait "$pid" 2>/dev/null || true; }
trap cleanup EXIT
for _ in $(seq 1 100); do
  if bash -c 'echo >/dev/tcp/127.0.0.1/18181' 2>/dev/null; then
    break
  fi
  sleep 0.1
done
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
