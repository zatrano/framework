<p align="center">
  <strong>ZATRANO</strong>
</p>

<p align="center">
  <a href="https://github.com/zatrano/framework/actions"><img src="https://github.com/zatrano/framework/actions/workflows/tests.yml/badge.svg" alt="Tests"></a>
  <a href="https://github.com/zatrano/framework/actions"><img src="https://github.com/zatrano/framework/actions/workflows/static-analysis.yml/badge.svg" alt="Static Analysis"></a>
  <a href="https://github.com/zatrano/framework/actions"><img src="https://github.com/zatrano/framework/actions/workflows/coding-style.yml/badge.svg" alt="Coding Style"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License"></a>
  <a href="VERSION"><img src="https://img.shields.io/badge/version-0.1.8-green.svg" alt="Version"></a>
</p>

## About ZATRANO

ZATRANO is a Go web application framework with expressive, elegant syntax. We believe development must be an enjoyable and creative experience to be truly fulfilling. ZATRANO takes the pain out of development by easing common tasks used in many web projects, such as:

- Simple, fast routing engine
- Powerful application container and service providers
- Multiple back-ends for session and cache storage
- Expressive database query builder and ORM
- Database agnostic schema migrations
- Robust background job processing
- Real-time event broadcasting
- Validation, authentication, authorization, mail, and notifications

ZATRANO keeps Go’s performance and clarity while giving you an opinionated application structure and the tools required for large, robust applications.

## Learning ZATRANO

You can learn more by reading the [ZATRANO documentation](https://zatrano.com/docs).

Install a tagged release as a Go module:

```bash
go get github.com/zatrano/framework@latest
go get github.com/zatrano/framework@v0.1.8
```

Or clone the repository and run the application skeleton:

```bash
git clone https://github.com/zatrano/framework.git
cd framework
cp .env.example .env
go mod tidy
go run ./cmd/zatrano key:generate
go run ./cmd/zatrano serve
```

Then open [http://localhost:8080](http://localhost:8080).

## Contributing

Thank you for considering contributing to the ZATRANO framework! Please open an issue or pull request on GitHub. Keep changes focused, include tests when practical, and follow the existing code style (`gofmt`, clear package boundaries).

## Code of Conduct

In order to ensure that the ZATRANO community is welcoming to all, please be respectful in issues, pull requests, and discussions. Harassment and discrimination are not tolerated.

## Security Vulnerabilities

If you discover a security vulnerability within ZATRANO, please send an e-mail to Serhan KARAKOÇ via [serhankarakoc@zatrano.com](mailto:serhankarakoc@zatrano.com). All security vulnerabilities will be promptly addressed.

## License

The ZATRANO framework is open-sourced software licensed under the [MIT license](LICENSE).

Copyright (c) 2026 Serhan KARAKOÇ.
