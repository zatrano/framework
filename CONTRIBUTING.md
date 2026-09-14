# Contributing

Thank you for considering contributing to the ZATRANO framework.

## Guidelines

- Keep pull requests focused on a single concern.
- Match existing naming, package layout, and coding style.
- Run `gofmt` on changed Go files.
- Add or update tests when behavior changes.
- Do not commit secrets, `.env` files, or local databases.

## Development

```bash
cp .env.example .env
go mod tidy
go run ./cmd/zatrano key:generate
go test ./...
```

## Pull Requests

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to your fork
5. Open a pull request against `main`

## Code of Conduct

Be respectful in issues, pull requests, and discussions. Harassment and discrimination are not tolerated.

## Security

Report security vulnerabilities privately to [serhankarakoc@gmail.com](mailto:serhankarakoc@gmail.com). See [.github/SECURITY.md](.github/SECURITY.md).

## Release cadence

Patch and minor versions are published **at most once per week**. The only exception is a tagged security fix: the CHANGELOG entry must include `[security]`, which is the only trigger for this exception.

Every release note must include a one-line `Neden şimdi:` reason. Do not add that line to historical entries; it is required from the next tagged release forward.

Releases are created **only** with `scripts/release.sh`. Do not run `git tag` by hand. `scripts/release.sh --dry-run <tag>` prints the validation steps.
