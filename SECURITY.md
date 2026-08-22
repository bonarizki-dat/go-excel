# Security Policy

## Supported Versions

Only the latest tagged release of `github.com/bonarizki-dat/go-excel` receives
security fixes. This project is pre-1.0 (`v0.x`), so no long-term support
branches are maintained.

| Version | Supported          |
| ------- | ------------------ |
| Latest  | :white_check_mark: |
| Older   | :x:                |

## Reporting a Vulnerability

Please report suspected security vulnerabilities privately, using GitHub's
[private vulnerability reporting](https://docs.github.com/en/code-security/security-advisories/guidance-on-reporting-and-writing/privately-reporting-a-security-vulnerability)
feature on this repository (Security tab -> "Report a vulnerability").

Do not open a public issue for suspected vulnerabilities.

Please include, where possible:

- A description of the vulnerability and its potential impact.
- Steps to reproduce, or a minimal code sample.
- The affected version(s) of this library and of `go.mod` dependencies
  (in particular `github.com/xuri/excelize/v2`, since much of this library's
  attack surface is inherited from it).

We aim to acknowledge new reports within 5 business days and to release a fix
or mitigation as soon as reasonably possible after confirming the issue.

## Dependency Vulnerabilities

This repository runs [`govulncheck`](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck)
in CI on every push and pull request, and Dependabot opens weekly pull
requests for outdated Go modules and GitHub Actions. You can run the same
check locally with:

```bash
make vulncheck
```
