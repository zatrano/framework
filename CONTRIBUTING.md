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
5. Open a pull request against `v2-dev` (active line) or `main`

## Code of Conduct

Be respectful in issues, pull requests, and discussions. Harassment and discrimination are not tolerated.

## Security

Report security vulnerabilities privately to [serhankarakoc@gmail.com](mailto:serhankarakoc@gmail.com). See [.github/SECURITY.md](.github/SECURITY.md).
