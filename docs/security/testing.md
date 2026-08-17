# Security testing

Commands for maintainers. Run from the repository root.

## Fast suite

```bash
go vet ./...
go test ./packages/database/query/ ./packages/session/ ./packages/auth/ ./packages/middleware/... -count=1
go test -C packages/mongo -count=1
go test ./packages/database/query/ -run=^$ -fuzz=FuzzSanitizeIdentifier -fuzztime=15s
```

## Race detector

```bash
go test -race ./packages/session/ ./packages/auth/ ./packages/ratelimit/ ./packages/middleware/... -count=1
```

## Static / vulnerability scanners

```bash
go install honnef.co/go/tools/cmd/staticcheck@2025.1.1
staticcheck ./...

go install golang.org/x/vuln/cmd/govulncheck@v1.1.4
govulncheck ./...

go install github.com/securego/gosec/v2/cmd/gosec@v2.22.2
gosec -quiet -exclude=G301,G306,G505,G104 -exclude-generated ./...
```

CI runs these via `.github/workflows/security.yml`.

## OWASP ZAP (local only)

1. Start the security demo:

```bash
go run ./tests/securitydemo
# listens on :18080
```

2. Baseline scan (Docker):

```bash
docker run --rm --network host zaproxy/zap-stable zap-baseline.py -t http://127.0.0.1:18080 -r zap-report.html
```

Do **not** point ZAP at production systems.

## Full audit artifacts

* `SECURITY_AUDIT.md` — architecture / threat model (Phase 0)
* `SECURITY_REPORT.md` — findings, fixes, remaining risks
