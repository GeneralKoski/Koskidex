# Security Policy

## Supported Versions

Koskidex is under active development. Security fixes are applied to the latest
release on the `main` branch.

## Reporting a Vulnerability

Please **do not** open public GitHub issues for security vulnerabilities.

Instead, report them privately via [GitHub Security Advisories](https://github.com/GeneralKoski/Koskidex/security/advisories/new)
or by email to the maintainer. Include:

- A description of the vulnerability and its impact.
- Steps to reproduce (proof of concept if possible).
- Affected version/commit.

You can expect an initial response within 72 hours. Once the issue is
confirmed, a fix will be prepared and a coordinated disclosure timeline agreed.

## Hardening Recommendations

When deploying Koskidex in production:

- Always set `KOSKIDEX_API_KEY` to require Bearer-token authentication.
- Terminate TLS in front of the service (reverse proxy) or enable built-in TLS.
- Enable rate limiting via `--rate-limit` to mitigate abuse.
- Restrict the data directory permissions; WAL and snapshot files contain all
  indexed data in plaintext.
- Do not expose the management endpoints (`/indexes`) to untrusted networks.
